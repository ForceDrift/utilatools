/*
Copyright 2018 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pdfcpu

import (
	"bytes"
	"sort"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	pdffont "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/primitives"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

func optimizeContentStreamUsage(ctx *model.Context, sd *types.StreamDict, objNr int) (*types.IndirectRef, error) {
	f := ctx.Optimize.ContentStreamCache
	if len(f) == 0 {
		f[objNr] = sd
		return nil, nil
	}

	if f[objNr] != nil {
		return nil, nil
	}

	cachedObjNrs := []int{}
	for objNr, sd1 := range f {
		if *sd1.StreamLength == *sd.StreamLength {
			cachedObjNrs = append(cachedObjNrs, objNr)
		}
	}
	if len(cachedObjNrs) == 0 {
		f[objNr] = sd
		return nil, nil
	}

	for _, objNr := range cachedObjNrs {
		sd1 := f[objNr]
		if bytes.Equal(sd.Raw, sd1.Raw) {
			ir := types.NewIndirectRef(objNr, 0)
			ctx.IncrementRefCount(ir)
			return ir, nil
		}
	}

	f[objNr] = sd
	return nil, nil
}

func removeEmptyContentStreams(ctx *model.Context, pageDict types.Dict, obj types.Object, pageObjNumber int) error {
	var contentArr types.Array

	if ir, ok := obj.(types.IndirectRef); ok {

		objNr := ir.ObjectNumber.Value()
		entry, found := ctx.FindTableEntry(objNr, ir.GenerationNumber.Value())
		if !found {
			return errors.Errorf("removeEmptyContentStreams: obj#:%d illegal indRef for Contents\n", pageObjNumber)
		}

		contentStreamDict, ok := entry.Object.(types.StreamDict)
		if ok {
			if err := contentStreamDict.Decode(); err != nil {
				return err
			}
			if len(contentStreamDict.Content) == 0 {
				pageDict.Delete("Contents")
			}
			return nil
		}

		contentArr, ok = entry.Object.(types.Array)
		if !ok {
			return errors.Errorf("removeEmptyContentStreams: obj#:%d page content entry neither stream dict nor array.\n", pageObjNumber)
		}

	} else if contentArr, ok = obj.(types.Array); !ok {
		return errors.Errorf("removeEmptyContentStreams: obj#:%d corrupt page content array\n", pageObjNumber)
	}

	var newContentArr types.Array

	for _, c := range contentArr {

		ir, ok := c.(types.IndirectRef)
		if !ok {
			return errors.Errorf("removeEmptyContentStreams: obj#:%d corrupt page content array entry\n", pageObjNumber)
		}

		objNr := ir.ObjectNumber.Value()
		entry, found := ctx.FindTableEntry(objNr, ir.GenerationNumber.Value())
		if !found {
			return errors.Errorf("removeEmptyContentStreams: obj#:%d illegal indRef for Contents\n", pageObjNumber)
		}

		contentStreamDict, ok := entry.Object.(types.StreamDict)
		if !ok {
			return errors.Errorf("identifyPageContent: obj#:%d page content entry is no stream dict\n", pageObjNumber)
		}

		if err := contentStreamDict.Decode(); err != nil {
			return err
		}
		if len(contentStreamDict.Content) > 0 {
			newContentArr = append(newContentArr, c)
		}
	}

	pageDict["Contents"] = newContentArr

	return nil
}

func optimizePageContent(ctx *model.Context, pageDict types.Dict, pageObjNumber int) error {
	o, found := pageDict.Find("Contents")
	if !found {
		return nil
	}

	if err := removeEmptyContentStreams(ctx, pageDict, o, pageObjNumber); err != nil {
		return err
	}

	o, found = pageDict.Find("Contents")
	if !found {
		return nil
	}

	if !ctx.OptimizeDuplicateContentStreams {
		return nil
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("identifyPageContent begin")
	}

	var contentArr types.Array

	if ir, ok := o.(types.IndirectRef); ok {

		objNr := ir.ObjectNumber.Value()
		entry, found := ctx.FindTableEntry(objNr, ir.GenerationNumber.Value())
		if !found {
			return errors.Errorf("identifyPageContent: obj#:%d illegal indRef for Contents\n", pageObjNumber)
		}

		contentStreamDict, ok := entry.Object.(types.StreamDict)
		if ok {
			ir, err := optimizeContentStreamUsage(ctx, &contentStreamDict, objNr)
			if err != nil {
				return err
			}
			if ir != nil {
				pageDict["Contents"] = *ir
			}
			contentStreamDict.IsPageContent = true
			entry.Object = contentStreamDict
			if log.OptimizeEnabled() {
				log.Optimize.Printf("identifyPageContent end: ok obj#%d\n", objNr)
			}
			return nil
		}

		contentArr, ok = entry.Object.(types.Array)
		if !ok {
			return errors.Errorf("identifyPageContent: obj#:%d page content entry neither stream dict nor array.\n", pageObjNumber)
		}

	} else if contentArr, ok = o.(types.Array); !ok {
		return errors.Errorf("identifyPageContent: obj#:%d corrupt page content array\n", pageObjNumber)
	}

	// TODO Activate content array opimization as soon as we have a proper test file.

	_ = contentArr

	// for i, c := range contentArr {

	// 	ir, ok := c.(IndirectRef)
	// 	if !ok {
	// 		return errors.Errorf("identifyPageContent: obj#:%d corrupt page content array entry\n", pageObjNumber)
	// 	}

	// 	objNr := ir.ObjectNumber.Value()
	// 	entry, found := ctx.FindTableEntry(objNr, ir.GenerationNumber.Value())
	// 	if !found {
	// 		return errors.Errorf("identifyPageContent: obj#:%d illegal indRef for Contents\n", pageObjNumber)
	// 	}

	// 	contentStreamDict, ok := entry.Object.(StreamDict)
	// 	if !ok {
	// 		return errors.Errorf("identifyPageContent: obj#:%d page content entry is no stream dict\n", pageObjNumber)
	// 	}

	// 	ir1, err := optimizeContentStreamUsage(ctx, &contentStreamDict, objNr)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if ir1 != nil {
	// 		contentArr[i] = *ir1
	// 	}

	// 	contentStreamDict.IsPageContent = true
	// 	entry.Object = contentStreamDict
	// 	log.Optimize.Printf("identifyPageContent: ok obj#%d\n", ir.GenerationNumber.Value())
	// }

	if log.OptimizeEnabled() {
		log.Optimize.Println("identifyPageContent end")
	}

	return nil
}

// resourcesDictForPageDict returns the resource dict for a page dict if there is any.
func resourcesDictForPageDict(xRefTable *model.XRefTable, pageDict types.Dict, pageObjNumber int) (types.Dict, error) {
	o, found := pageDict.Find("Resources")
	if !found {
		if log.OptimizeEnabled() {
			log.Optimize.Printf("resourcesDictForPageDict end: No resources dict for page object %d, may be inherited\n", pageObjNumber)
		}
		return nil, nil
	}

	return xRefTable.DereferenceDict(o)
}

// handleDuplicateFontObject returns nil or the object number of the registered font if it matches this font.
func handleDuplicateFontObject(ctx *model.Context, fontDict types.Dict, fName, rName string, objNr, pageNr int) (*int, error) {
	// Get a slice of all font object numbers for font name.
	fontObjNrs, found := ctx.Optimize.Fonts[fName]
	if !found {
		// There is no registered font with fName.
		return nil, nil
	}

	// Get the set of font object numbers for pageNr.
	pageFonts := ctx.Optimize.PageFonts[pageNr]

	// Iterate over all registered font object numbers for font name.
	// Check if this font dict matches the font dict of each font object number.
	for _, fontObjNr := range fontObjNrs {

		// Get the font object from the lookup table.
		fontObject, ok := ctx.Optimize.FontObjects[fontObjNr]
		if !ok {
			continue
		}

		if log.OptimizeEnabled() {
			log.Optimize.Printf("handleDuplicateFontObject: comparing with fontDict Obj %d\n", fontObjNr)
		}

		// Check if the input fontDict matches the fontDict of this fontObject.
		ok, err := model.EqualFontDicts(fontObject.FontDict, fontDict, ctx.XRefTable)
		if err != nil {
			return nil, err
		}

		if !ok {
			// No match!
			continue
		}

		// We have detected a redundant font dict!
		if log.OptimizeEnabled() {
			log.Optimize.Printf("handleDuplicateFontObject: redundant fontObj#:%d basefont %s already registered with obj#:%d !\n", objNr, fName, fontObjNr)
		}

		// Register new page font with pageNr.
		// The font for font object number is used instead of objNr.
		pageFonts[fontObjNr] = true

		// Add the resource name of this duplicate font to the list of registered resource names.
		fontObject.AddResourceName(rName)

		// Register fontDict as duplicate.
		ctx.Optimize.DuplicateFonts[objNr] = fontDict

		// Return the fontObjectNumber that will be used instead of objNr.
		return &fontObjNr, nil
	}

	return nil, nil
}

func pageImages(ctx *model.Context, pageNr int) types.IntSet {
	pageImages := ctx.Optimize.PageImages[pageNr]
	if pageImages == nil {
		pageImages = types.IntSet{}
		ctx.Optimize.PageImages[pageNr] = pageImages
	}

	return pageImages
}

func pageFonts(ctx *model.Context, pageNr int) types.IntSet {
	pageFonts := ctx.Optimize.PageFonts[pageNr]
	if pageFonts == nil {
		pageFonts = types.IntSet{}
		ctx.Optimize.PageFonts[pageNr] = pageFonts
	}

	return pageFonts
}

func registerFontDictObjNr(ctx *model.Context, fName string, objNr int) {
	if log.OptimizeEnabled() {
		log.Optimize.Printf("optimizeFontResourcesDict: adding new font %s obj#%d\n", fName, objNr)
	}

	fontObjNrs, found := ctx.Optimize.Fonts[fName]
	if found {
		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeFontResourcesDict: appending %d to %s\n", objNr, fName)
		}
		ctx.Optimize.Fonts[fName] = append(fontObjNrs, objNr)
	} else {
		ctx.Optimize.Fonts[fName] = []int{objNr}
	}
}

// Get rid of redundant fonts for given fontResources dictionary.
func optimizeFontResourcesDict(ctx *model.Context, rDict types.Dict, pageNr, pageObjNumber int, rNamePrefix string) error {
	if log.OptimizeEnabled() {
		log.Optimize.Printf("optimizeFontResourcesDict begin: page=%d pageObjNumber=%d %s\nPageFonts=%v\n", pageNr, pageObjNumber, rDict, ctx.Optimize.PageFonts)
	}

	pageFonts := pageFonts(ctx, pageNr)

	// Iterate over font resource dict.
	for rName, v := range rDict {

		indRef, ok := v.(types.IndirectRef)
		if !ok {
			continue
		}

		objNr := int(indRef.ObjectNumber)

		qualifiedRName := rName
		if rNamePrefix != "" {
			qualifiedRName = rNamePrefix + "." + rName
		}

		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeFontResourcesDict: processing font: %s, obj#=%d\n", qualifiedRName, objNr)
		}

		if _, found := ctx.Optimize.FontObjects[objNr]; found {
			// This font has already been registered.
			//log.Optimize.Printf("optimizeFontResourcesDict: Fontobject %d already registered\n", objectNumber)
			pageFonts[objNr] = true
			continue
		}

		// We are dealing with a new font.
		fontDict, err := ctx.DereferenceFontDict(indRef)
		if err != nil {
			return err
		}
		if fontDict == nil {
			continue
		}

		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeFontResourcesDict: fontDict: %s\n", fontDict)
		}

		// Get the unique font name.
		prefix, fName, err := pdffont.Name(ctx.XRefTable, fontDict, objNr)
		if err != nil {
			return err
		}

		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeFontResourcesDict: baseFont: prefix=%s name=%s\n", prefix, fName)
		}

		// Check if fontDict is a duplicate and if so return the object number of the original.
		originalObjNr, err := handleDuplicateFontObject(ctx, fontDict, fName, qualifiedRName, objNr, pageNr)
		if err != nil {
			return err
		}

		if originalObjNr != nil {
			// We have identified a redundant fontDict!
			// Update font resource dict so that rName points to the original.
			ir := types.NewIndirectRef(*originalObjNr, 0)
			rDict[rName] = *ir
			ctx.IncrementRefCount(ir)
			continue
		}

		registerFontDictObjNr(ctx, fName, objNr)

		ctx.Optimize.FontObjects[objNr] =
			&model.FontObject{
				ResourceNames: []string{qualifiedRName},
				Prefix:        prefix,
				FontName:      fName,
				FontDict:      fontDict,
			}

		pageFonts[objNr] = true
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("optimizeFontResourcesDict end:")
	}

	return nil
}

// handleDuplicateImageObject returns nil or the object number of the registered image if it matches this image.
func handleDuplicateImageObject(ctx *model.Context, imageDict *types.StreamDict, resourceName string, objNr, pageNr int) (*int, error) {
	// Get the set of image object numbers for pageNr.
	pageImages := ctx.Optimize.PageImages[pageNr]

	// Process image dict, check if this is a duplicate.
	for imageObjNr, imageObject := range ctx.Optimize.ImageObjects {

		if log.OptimizeEnabled() {
			log.Optimize.Printf("handleDuplicateImageObject: comparing with imagedict Obj %d\n", imageObjNr)
		}

		// Check if the input imageDict matches the imageDict of this imageObject.
		ok, err := model.EqualStreamDicts(imageObject.ImageDict, imageDict, ctx.XRefTable)
		if err != nil {
			return nil, err
		}

		if !ok {
			// No match!
			continue
		}

		// We have detected a redundant image dict.
		if log.OptimizeEnabled() {
			log.Optimize.Printf("handleDuplicateImageObject: redundant imageObj#:%d already registered with obj#:%d !\n", objNr, imageObjNr)
		}

		// Register new page image for pageNr.
		// The image for image object number is used instead of objNr.
		pageImages[imageObjNr] = true

		// Add the resource name of this duplicate image to the list of registered resource names.
		imageObject.AddResourceName(pageNr, resourceName)

		// Register imageDict as duplicate.
		ctx.Optimize.DuplicateImages[objNr] = imageDict

		// Return the imageObjectNumber that will be used instead of objNr.
		return &imageObjNr, nil
	}

	return nil, nil
}

func optimizeXObjectImage(ctx *model.Context, osd *types.StreamDict, rNamePrefix, rName string, rDict types.Dict, objNr, pageNr int, pageImages types.IntSet) error {

	qualifiedRName := rName
	if rNamePrefix != "" {
		qualifiedRName = rNamePrefix + "." + rName
	}

	// Check if image is a duplicate and if so return the object number of the original.
	originalObjNr, err := handleDuplicateImageObject(ctx, osd, qualifiedRName, objNr, pageNr)
	if err != nil {
		return err
	}

	if originalObjNr != nil {
		// We have identified a redundant image!
		// Update xobject resource dict so that rName points to the original.
		ir := types.NewIndirectRef(*originalObjNr, 0)
		ctx.IncrementRefCount(ir)
		rDict[rName] = *ir
		return nil
	}

	// Register new image dict.
	if log.OptimizeEnabled() {
		log.Optimize.Printf("optimizeXObjectResourcesDict: adding new image obj#%d\n", objNr)
	}

	ctx.Optimize.ImageObjects[objNr] =
		&model.ImageObject{
			ResourceNames: map[int]string{pageNr: qualifiedRName},
			ImageDict:     osd,
		}

	pageImages[objNr] = true
	return nil
}

func optimizeXObjectForm(ctx *model.Context, sd *types.StreamDict, objNr int) (*types.IndirectRef, error) {

	f := ctx.Optimize.FormStreamCache
	if len(f) == 0 {
		f[objNr] = sd
		return nil, nil
	}

	if f[objNr] != nil {
		return nil, nil
	}

	cachedObjNrs := []int{}
	for objNr, sd1 := range f {
		if *sd1.StreamLength == *sd.StreamLength {
			cachedObjNrs = append(cachedObjNrs, objNr)
		}
	}
	if len(cachedObjNrs) == 0 {
		f[objNr] = sd
		return nil, nil
	}

	for _, objNr1 := range cachedObjNrs {
		sd1 := f[objNr1]
		ok, err := model.EqualStreamDicts(sd, sd1, ctx.XRefTable)
		if err != nil {
			return nil, err
		}
		if ok {
			ir := types.NewIndirectRef(objNr1, 0)
			ctx.IncrementRefCount(ir)
			return ir, nil
		}
	}

	f[objNr] = sd
	return nil, nil
}

func optimizeFormResources(ctx *model.Context, o types.Object, pageNr, pageObjNumber int, rName string, visitedRes []types.Object) error {
	d, err := ctx.DereferenceDict(o)
	if err != nil {
		return err
	}
	if d != nil {
		// Optimize image and font resources.
		if err = optimizeResources(ctx, d, pageNr, pageObjNumber, rName, visitedRes); err != nil {
			return err
		}
	}
	return nil
}

func visited(o types.Object, visited []types.Object) bool {
	for _, obj := range visited {
		if obj == o {
			return true
		}
	}
	return false
}

func optimizeForm(ctx *model.Context, osd *types.StreamDict, rNamePrefix, rName string, rDict types.Dict, objNr, pageNr, pageObjNumber int, vis []types.Object) error {

	ir, err := optimizeXObjectForm(ctx, osd, objNr)
	if err != nil {
		return err
	}

	if ir != nil {
		rDict[rName] = *ir
		return nil
	}

	o, found := osd.Find("Resources")
	if !found {
		return nil
	}

	indRef, ok := o.(types.IndirectRef)
	if ok {
		if visited(indRef, vis) {
			return nil
		}
		vis = append(vis, indRef)
	}

	qualifiedRName := rName
	if rNamePrefix != "" {
		qualifiedRName = rNamePrefix + "." + rName
	}

	return optimizeFormResources(ctx, o, pageNr, pageObjNumber, qualifiedRName, vis)
}

func optimizeXObjectResourcesDict(ctx *model.Context, rDict types.Dict, pageNr, pageObjNumber int, rNamePrefix string, vis []types.Object) error {
	if log.OptimizeEnabled() {
		log.Optimize.Printf("optimizeXObjectResourcesDict page#%dbegin: %s\n", pageObjNumber, rDict)
	}

	pageImages := pageImages(ctx, pageNr)

	for rName, v := range rDict {

		indRef, ok := v.(types.IndirectRef)
		if !ok {
			continue
		}

		if visited(indRef, vis) {
			continue
		}

		vis = append(vis, indRef)

		objNr := int(indRef.ObjectNumber)

		qualifiedRName := rName
		if rNamePrefix != "" {
			qualifiedRName = rNamePrefix + "." + rName
		}

		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeXObjectResourcesDict: processing XObject: %s, obj#=%d\n", qualifiedRName, objNr)
		}

		sd, err := ctx.DereferenceXObjectDict(indRef)
		if err != nil {
			return err
		}
		if sd == nil {
			continue
		}

		if err := ctx.DeleteDictEntry(sd.Dict, "PieceInfo"); err != nil {
			return err
		}

		if *sd.Subtype() == "Image" {
			if err := optimizeXObjectImage(ctx, sd, rNamePrefix, rName, rDict, objNr, pageNr, pageImages); err != nil {
				return err
			}
		}

		if *sd.Subtype() == "Form" {
			if err := optimizeForm(ctx, sd, rNamePrefix, rName, rDict, objNr, pageNr, pageObjNumber, vis); err != nil {
				return err
			}
		}

	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("optimizeXObjectResourcesDict end")
	}

	return nil
}

// Optimize given resource dictionary by removing redundant fonts and images.
func optimizeResources(ctx *model.Context, resourcesDict types.Dict, pageNr, pageObjNumber int, rNamePrefix string, visitedRes []types.Object) error {
	if log.OptimizeEnabled() {
		log.Optimize.Printf("optimizeResources begin: pageNr=%d pageObjNumber=%d\n", pageNr, pageObjNumber)
	}

	if resourcesDict == nil {
		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeResources end: No resources dict available")
		}
		return nil
	}

	// Process Font resource dict, get rid of redundant fonts.
	o, found := resourcesDict.Find("Font")
	if found {

		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return err
		}

		if d == nil {
			return errors.Errorf("pdfcpu: optimizeResources: font resource dict is null for page %d pageObj %d\n", pageNr, pageObjNumber)
		}

		if err = optimizeFontResourcesDict(ctx, d, pageNr, pageObjNumber, rNamePrefix); err != nil {
			return err
		}

	}

	// Note: An optional ExtGState resource dict may contain binary content in the following entries: "SMask", "HT".

	// Process XObject resource dict, get rid of redundant images.
	o, found = resourcesDict.Find("XObject")
	if found {

		d, err := ctx.DereferenceDict(o)
		if err != nil {
			return err
		}

		if d == nil {
			return errors.Errorf("pdfcpu: optimizeResources: xobject resource dict is null for page %d pageObj %d\n", pageNr, pageObjNumber)
		}

		if err = optimizeXObjectResourcesDict(ctx, d, pageNr, pageObjNumber, rNamePrefix, visitedRes); err != nil {
			return err
		}

	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("optimizeResources end")
	}

	return nil
}

// Process the resources dictionary for given page number and optimize by removing redundant resources.
func parseResourcesDict(ctx *model.Context, pageDict types.Dict, pageNr, pageObjNumber int) error {
	if ctx.Optimize.Cache[pageObjNumber] {
		return nil
	}
	ctx.Optimize.Cache[pageObjNumber] = true

	// The logical pageNr is pageNr+1.
	if log.OptimizeEnabled() {
		log.Optimize.Printf("parseResourcesDict begin page: %d, object:%d\n", pageNr+1, pageObjNumber)
	}

	// Get resources dict for this page.
	d, err := resourcesDictForPageDict(ctx.XRefTable, pageDict, pageObjNumber)
	if err != nil {
		return err
	}

	// dict may be nil for inherited resource dicts.
	if d != nil {

		// Optimize image and font resources.
		if err = optimizeResources(ctx, d, pageNr, pageObjNumber, "", []types.Object{}); err != nil {
			return err
		}

	}

	if log.OptimizeEnabled() {
		log.Optimize.Printf("parseResourcesDict end page: %d, object:%d\n", pageNr+1, pageObjNumber)
	}

	return nil
}

// Iterate over all pages and optimize content & resources.
func parsePagesDict(ctx *model.Context, pagesDict types.Dict, pageNr int) (int, error) {
	// TODO Integrate resource consolidation based on content stream requirements.

	count, found := pagesDict.Find("Count")
	if !found {
		return pageNr, errors.New("pdfcpu: parsePagesDict: missing Count")
	}

	if log.OptimizeEnabled() {
		log.Optimize.Printf("parsePagesDict begin (next page=%d has %s pages): %s\n", pageNr+1, count.(types.Integer), pagesDict)
	}

	ctx.Optimize.Cache = map[int]bool{}

	// Iterate over page tree.
	o, found := pagesDict.Find("Kids")
	if !found {
		return pageNr, errors.Errorf("pdfcpu: corrupt \"Kids\" entry %s", pagesDict)
	}

	kids, err := ctx.DereferenceArray(o)
	if err != nil || kids == nil {
		return pageNr, errors.Errorf("pdfcpu: corrupt \"Kids\" entry: %s", pagesDict)
	}

	for _, v := range kids {

		// Dereference next page node dict.
		ir, _ := v.(types.IndirectRef)

		if log.OptimizeEnabled() {
			log.Optimize.Printf("parsePagesDict PageNode: %s\n", ir)
		}

		d, err := ctx.DereferencePageNodeDict(ir)
		if err != nil {
			return 0, errors.Wrap(err, "parsePagesDict: can't locate Pagedict or Pagesdict")
		}

		dictType := d.Type()

		// Note: Resource dicts may be inherited.

		if *dictType == "Pages" {

			// Recurse over pagetree and optimize resources.
			pageNr, err = parsePagesDict(ctx, d, pageNr)
			if err != nil {
				return 0, err
			}

			continue
		}

		// Process page dict.

		if err = optimizePageContent(ctx, d, int(ir.ObjectNumber)); err != nil {
			return 0, err
		}

		if err := ctx.DeleteDictEntry(d, "PieceInfo"); err != nil {
			return 0, err
		}

		// Parse and optimize resource dict for one page.
		if err = parseResourcesDict(ctx, d, pageNr, int(ir.ObjectNumber)); err != nil {
			return 0, err
		}

		pageNr++
	}

	if log.OptimizeEnabled() {
		log.Optimize.Printf("parsePagesDict end: %s\n", pagesDict)
	}

	return pageNr, nil
}

func traverse(xRefTable *model.XRefTable, value types.Object, duplObjs types.IntSet) error {
	if indRef, ok := value.(types.IndirectRef); ok {
		duplObjs[int(indRef.ObjectNumber)] = true
		o, err := xRefTable.Dereference(indRef)
		if err != nil {
			return err
		}
		traverseObjectGraphAndMarkDuplicates(xRefTable, o, duplObjs)
	}
	if d, ok := value.(types.Dict); ok {
		traverseObjectGraphAndMarkDuplicates(xRefTable, d, duplObjs)
	}
	if sd, ok := value.(types.StreamDict); ok {
		traverseObjectGraphAndMarkDuplicates(xRefTable, sd, duplObjs)
	}
	if a, ok := value.(types.Array); ok {
		traverseObjectGraphAndMarkDuplicates(xRefTable, a, duplObjs)
	}

	return nil
}

// Traverse the object graph for a Object and mark all objects as potential duplicates.
func traverseObjectGraphAndMarkDuplicates(xRefTable *model.XRefTable, obj types.Object, duplObjs types.IntSet) error {
	if log.OptimizeEnabled() {
		log.Optimize.Printf("traverseObjectGraphAndMarkDuplicates begin type=%T\n", obj)
	}

	switch x := obj.(type) {

	case types.Dict:
		if log.OptimizeEnabled() {
			log.Optimize.Println("traverseObjectGraphAndMarkDuplicates: dict")
		}
		for _, value := range x {
			if err := traverse(xRefTable, value, duplObjs); err != nil {
				return err
			}
		}

	case types.StreamDict:
		if log.OptimizeEnabled() {
			log.Optimize.Println("traverseObjectGraphAndMarkDuplicates: streamDict")
		}
		for _, value := range x.Dict {
			if err := traverse(xRefTable, value, duplObjs); err != nil {
				return err
			}
		}

	case types.Array:
		if log.OptimizeEnabled() {
			log.Optimize.Println("traverseObjectGraphAndMarkDuplicates: arr")
		}
		for _, value := range x {
			if err := traverse(xRefTable, value, duplObjs); err != nil {
				return err
			}
		}
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("traverseObjectGraphAndMarkDuplicates end")
	}

	return nil
}

// Identify and mark all potential duplicate objects.
func calcRedundantObjects(ctx *model.Context) error {
	if log.OptimizeEnabled() {
		log.Optimize.Println("calcRedundantObjects begin")
	}

	for i, fontDict := range ctx.Optimize.DuplicateFonts {
		ctx.Optimize.DuplicateFontObjs[i] = true
		// Identify and mark all involved potential duplicate objects for a redundant font.
		if err := traverseObjectGraphAndMarkDuplicates(ctx.XRefTable, fontDict, ctx.Optimize.DuplicateFontObjs); err != nil {
			return err
		}
	}

	for i, sd := range ctx.Optimize.DuplicateImages {
		ctx.Optimize.DuplicateImageObjs[i] = true
		// Identify and mark all involved potential duplicate objects for a redundant image.
		if err := traverseObjectGraphAndMarkDuplicates(ctx.XRefTable, *sd, ctx.Optimize.DuplicateImageObjs); err != nil {
			return err
		}
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("calcRedundantObjects end")
	}

	return nil
}

// Iterate over all pages and optimize resources.
// Get rid of duplicate embedded fonts and images.
func optimizeFontAndImages(ctx *model.Context) error {
	if log.OptimizeEnabled() {
		log.Optimize.Println("optimizeFontAndImages begin")
	}

	// Get a reference to the PDF indirect reference of the page tree root dict.
	indRefPages, err := ctx.Pages()
	if err != nil {
		return err
	}

	// Dereference and get a reference to the page tree root dict.
	pageTreeRootDict, err := ctx.XRefTable.DereferenceDict(*indRefPages)
	if err != nil {
		return err
	}

	// Detect the number of pages of this PDF file.
	pageCount := pageTreeRootDict.IntEntry("Count")
	if pageCount == nil {
		return errors.New("pdfcpu: optimizeFontAndImagess: missing \"Count\" in page root dict")
	}

	// If PageCount already set by validation doublecheck.
	if ctx.PageCount > 0 && ctx.PageCount != *pageCount {
		return errors.New("pdfcpu: optimizeFontAndImagess: unexpected page root dict pageCount discrepancy")
	}

	// If we optimize w/o prior validation, set PageCount.
	if ctx.PageCount == 0 {
		ctx.PageCount = *pageCount
	}

	// Prepare optimization environment.
	ctx.Optimize.PageFonts = make([]types.IntSet, ctx.PageCount)
	ctx.Optimize.PageImages = make([]types.IntSet, ctx.PageCount)

	// Iterate over page dicts and optimize resources.
	_, err = parsePagesDict(ctx, pageTreeRootDict, 0)
	if err != nil {
		return err
	}

	ctx.Optimize.ContentStreamCache = map[int]*types.StreamDict{}
	ctx.Optimize.FormStreamCache = map[int]*types.StreamDict{}

	// Identify all duplicate objects.
	if err = calcRedundantObjects(ctx); err != nil {
		return err
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("optimizeFontAndImages end")
	}

	return nil
}

// Return stream length for font file object.
func streamLengthFontFile(xRefTable *model.XRefTable, indirectRef *types.IndirectRef) (*int64, error) {
	if log.OptimizeEnabled() {
		log.Optimize.Println("streamLengthFontFile begin")
	}

	objectNumber := indirectRef.ObjectNumber

	sd, _, err := xRefTable.DereferenceStreamDict(*indirectRef)
	if err != nil {
		return nil, err
	}

	if sd == nil || (*sd).StreamLength == nil {
		return nil, errors.Errorf("pdfcpu: streamLengthFontFile: fontFile Streamlength is nil for object %d\n", objectNumber)
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("streamLengthFontFile end")
	}

	return (*sd).StreamLength, nil
}

// Calculate amount of memory used by embedded fonts for stats.
func calcEmbeddedFontsMemoryUsage(ctx *model.Context) error {
	if log.OptimizeEnabled() {
		log.Optimize.Printf("calcEmbeddedFontsMemoryUsage begin: %d fontObjects\n", len(ctx.Optimize.FontObjects))
	}

	fontFileIndRefs := map[types.IndirectRef]bool{}

	var objectNumbers []int

	// Sorting unnecessary.
	for k := range ctx.Optimize.FontObjects {
		objectNumbers = append(objectNumbers, k)
	}
	sort.Ints(objectNumbers)

	// Iterate over all embedded font objects and record font file references.
	for _, objectNumber := range objectNumbers {

		fontObject := ctx.Optimize.FontObjects[objectNumber]

		// Only embedded fonts have binary data.
		if !fontObject.Embedded() {
			continue
		}

		if err := processFontFilesForFontDict(ctx.XRefTable, fontObject.FontDict, objectNumber, fontFileIndRefs); err != nil {
			return err
		}
	}

	// Iterate over font file references and calculate total font size.
	for ir := range fontFileIndRefs {
		streamLength, err := streamLengthFontFile(ctx.XRefTable, &ir)
		if err != nil {
			return err
		}
		ctx.Read.BinaryFontSize += *streamLength
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("calcEmbeddedFontsMemoryUsage end")
	}

	return nil
}

// fontDescriptorFontFileIndirectObjectRef returns the indirect object for the font file for given font descriptor.
func fontDescriptorFontFileIndirectObjectRef(fontDescriptorDict types.Dict) *types.IndirectRef {
	if log.OptimizeEnabled() {
		log.Optimize.Println("fontDescriptorFontFileIndirectObjectRef begin")
	}

	ir := fontDescriptorDict.IndirectRefEntry("FontFile")

	if ir == nil {
		ir = fontDescriptorDict.IndirectRefEntry("FontFile2")
	}

	if ir == nil {
		ir = fontDescriptorDict.IndirectRefEntry("FontFile3")
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("FontDescriptorFontFileIndirectObjectRef end")
	}

	return ir
}

func trivialFontDescriptor(xRefTable *model.XRefTable, fontDict types.Dict, objNr int) (types.Dict, error) {
	o, ok := fontDict.Find("FontDescriptor")
	if !ok {
		return nil, nil
	}

	// fontDescriptor directly available.

	d, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return nil, err
	}

	if d == nil {
		return nil, errors.Errorf("pdfcpu: trivialFontDescriptor: FontDescriptor is null for font object %d\n", objNr)
	}

	if d.Type() != nil && *d.Type() != "FontDescriptor" {
		return nil, errors.Errorf("pdfcpu: trivialFontDescriptor: FontDescriptor dict incorrect dict type for font object %d\n", objNr)
	}

	return d, nil
}

// FontDescriptor gets the font descriptor for this font.
func fontDescriptor(xRefTable *model.XRefTable, fontDict types.Dict, objNr int) (types.Dict, error) {
	if log.OptimizeEnabled() {
		log.Optimize.Println("fontDescriptor begin")
	}

	d, err := trivialFontDescriptor(xRefTable, fontDict, objNr)
	if err != nil {
		return nil, err
	}
	if d != nil {
		return d, nil
	}

	// Try to access a fontDescriptor in a Descendent font for Type0 fonts.

	o, ok := fontDict.Find("DescendantFonts")
	if !ok {
		//logErrorOptimize.Printf("FontDescriptor: Neither FontDescriptor nor DescendantFonts for font object %d\n", objectNumber)
		return nil, nil
	}

	// A descendant font is contained in an array of size 1.

	a, err := xRefTable.DereferenceArray(o)
	if err != nil || a == nil {
		return nil, errors.Errorf("pdfcpu: fontDescriptor: DescendantFonts: IndirectRef or Array wth length 1 expected for font object %d\n", objNr)
	}
	if len(a) > 1 {
		return nil, errors.Errorf("pdfcpu: fontDescriptor: DescendantFonts Array length > 1 %v\n", a)
	}

	// dict is the fontDict of the descendant font.
	d, err = xRefTable.DereferenceDict(a[0])
	if err != nil {
		return nil, errors.Errorf("pdfcpu: fontDescriptor: No descendant font dict for %v\n", a)
	}
	if d == nil {
		return nil, errors.Errorf("pdfcpu: fontDescriptor: descendant font dict is null for %v\n", a)
	}

	if *d.Type() != "Font" {
		return nil, errors.Errorf("pdfcpu: fontDescriptor: font dict with incorrect dict type for %v\n", d)
	}

	o, ok = d.Find("FontDescriptor")
	if !ok {
		log.Optimize.Printf("fontDescriptor: descendant font not embedded %s\n", d)
		return nil, nil
	}

	d, err = xRefTable.DereferenceDict(o)
	if err != nil {
		return nil, errors.Errorf("pdfcpu: fontDescriptor: No FontDescriptor dict for font object %d\n", objNr)
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("fontDescriptor end")
	}

	return d, nil
}

// Record font file objects referenced by this fonts font descriptor for stats and size calculation.
func processFontFilesForFontDict(xRefTable *model.XRefTable, fontDict types.Dict, objectNumber int, indRefsMap map[types.IndirectRef]bool) error {
	if log.OptimizeEnabled() {
		log.Optimize.Println("processFontFilesForFontDict begin")
	}

	// Note:
	// "ToUnicode" is also an entry containing binary content that could be inspected for duplicate content.

	d, err := fontDescriptor(xRefTable, fontDict, objectNumber)
	if err != nil {
		return err
	}

	if d != nil {
		if ir := fontDescriptorFontFileIndirectObjectRef(d); ir != nil {
			indRefsMap[*ir] = true
		}
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("processFontFilesForFontDict end")
	}

	return nil
}

// Calculate amount of memory used by duplicate embedded fonts for stats.
func calcRedundantEmbeddedFontsMemoryUsage(ctx *model.Context) error {
	if log.OptimizeEnabled() {
		log.Optimize.Println("calcRedundantEmbeddedFontsMemoryUsage begin")
	}

	fontFileIndRefs := map[types.IndirectRef]bool{}

	// Iterate over all duplicate fonts and record font file references.
	for objectNumber, fontDict := range ctx.Optimize.DuplicateFonts {

		// Duplicate Fonts have to be embedded, so no check here.
		if err := processFontFilesForFontDict(ctx.XRefTable, fontDict, objectNumber, fontFileIndRefs); err != nil {
			return err
		}

	}

	// Iterate over font file references and calculate total font size.
	for ir := range fontFileIndRefs {

		streamLength, err := streamLengthFontFile(ctx.XRefTable, &ir)
		if err != nil {
			return err
		}

		ctx.Read.BinaryFontDuplSize += *streamLength
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("calcRedundantEmbeddedFontsMemoryUsage end")
	}

	return nil
}

// Calculate amount of memory used by embedded fonts and duplicate embedded fonts for stats.
func calcFontBinarySizes(ctx *model.Context) error {
	if log.OptimizeEnabled() {
		log.Optimize.Println("calcFontBinarySizes begin")
	}

	if err := calcEmbeddedFontsMemoryUsage(ctx); err != nil {
		return err
	}

	if err := calcRedundantEmbeddedFontsMemoryUsage(ctx); err != nil {
		return err
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("calcFontBinarySizes end")
	}

	return nil
}

// Calculate amount of memory used by images and duplicate images for stats.
func calcImageBinarySizes(ctx *model.Context) {
	if log.OptimizeEnabled() {
		log.Optimize.Println("calcImageBinarySizes begin")
	}

	// Calc memory usage for images.
	for _, imageObject := range ctx.Optimize.ImageObjects {
		ctx.Read.BinaryImageSize += *imageObject.ImageDict.StreamLength
	}

	// Calc memory usage for duplicate images.
	for _, imageDict := range ctx.Optimize.DuplicateImages {
		ctx.Read.BinaryImageDuplSize += *imageDict.StreamLength
	}

	if log.OptimizeEnabled() {
		log.Optimize.Println("calcImageBinarySizes end")
	}
}

// Calculate memory usage of binary data for stats.
func calcBinarySizes(ctx *model.Context) error {
	if log.OptimizeEnabled() {
		log.Optimize.Println("calcBinarySizes begin")
	}

	// Calculate font memory usage for stats.
	if err := calcFontBinarySizes(ctx); err != nil {
		return err
	}

	// Calculate image memory usage for stats.
	calcImageBinarySizes(ctx)

	// Note: Content streams also represent binary content.

	if log.OptimizeEnabled() {
		log.Optimize.Println("calcBinarySizes end")
	}

	return nil
}

func fixDeepDict(ctx *model.Context, d types.Dict) error {
	for k, v := range d {
		ir, err := fixDeepObject(ctx, v)
		if err != nil {
			return err
		}
		if ir != nil {
			d[k] = *ir
		}
	}

	return nil
}

func fixDeepArray(ctx *model.Context, a types.Array) error {
	for i, v := range a {
		ir, err := fixDeepObject(ctx, v)
		if err != nil {
			return err
		}
		if ir != nil {
			a[i] = *ir
		}
	}

	return nil
}

func fixDirectObject(ctx *model.Context, o types.Object) error {
	switch o := o.(type) {
	case types.Dict:
		for k, v := range o {
			ir, err := fixDeepObject(ctx, v)
			if err != nil {
				return err
			}
			if ir != nil {
				o[k] = *ir
			}
		}
	case types.Array:
		for i, v := range o {
			ir, err := fixDeepObject(ctx, v)
			if err != nil {
				return err
			}
			if ir != nil {
				o[i] = *ir
			}
		}
	}

	return nil
}

func fixIndirectObject(ctx *model.Context, ir *types.IndirectRef) error {
	objNr := int(ir.ObjectNumber)

	if ctx.Optimize.Cache[objNr] {
		return nil
	}
	ctx.Optimize.Cache[objNr] = true

	entry, found := ctx.Find(objNr)
	if !found {
		return nil
	}

	if entry.Free {
		// This is a reference to a free object that needs to be fixed.

		//fmt.Printf("fixNullObject: #%d g%d\n", objNr, genNr)

		if ctx.Optimize.NullObjNr == nil {
			nr, err := ctx.InsertObject(nil)
			if err != nil {
				return err
			}
			ctx.Optimize.NullObjNr = &nr
		}

		ir.ObjectNumber = types.Integer(*ctx.Optimize.NullObjNr)

		return nil
	}

	var err error

	switch o := entry.Object.(type) {

	case types.Dict:
		err = fixDeepDict(ctx, o)

	case types.StreamDict:
		err = fixDeepDict(ctx, o.Dict)

	case types.Array:
		err = fixDeepArray(ctx, o)

	}

	return err
}

func fixDeepObject(ctx *model.Context, o types.Object) (*types.IndirectRef, error) {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return nil, fixDirectObject(ctx, o)
	}

	err := fixIndirectObject(ctx, &ir)
	return &ir, err
}

func fixReferencesToFreeObjects(ctx *model.Context) error {
	return fixDirectObject(ctx, ctx.RootDict)
}

func CacheFormFonts(ctx *model.Context) error {

	d, err := primitives.FormFontResDict(ctx.XRefTable)
	if err != nil {
		return err
	}

	// Iterate over font resource dict.
	for rName, v := range d {

		indRef, ok := v.(types.IndirectRef)
		if !ok {
			continue
		}

		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeFontResourcesDict: processing font: %s, %s\n", rName, indRef)
		}

		objNr := int(indRef.ObjectNumber)

		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeFontResourcesDict: objectNumber = %d\n", objNr)
		}

		fontDict, err := ctx.DereferenceFontDict(indRef)
		if err != nil {
			return err
		}
		if fontDict == nil {
			continue
		}

		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeFontResourcesDict: fontDict: %s\n", fontDict)
		}

		// Get the unique font name.
		prefix, fName, err := pdffont.Name(ctx.XRefTable, fontDict, objNr)
		if err != nil {
			return err
		}

		if log.OptimizeEnabled() {
			log.Optimize.Printf("optimizeFontResourcesDict: baseFont: prefix=%s name=%s\n", prefix, fName)
		}

		registerFontDictObjNr(ctx, fName, objNr)

		ctx.Optimize.FormFontObjects[objNr] =
			&model.FontObject{
				ResourceNames: []string{rName},
				Prefix:        prefix,
				FontName:      fName,
				FontDict:      fontDict,
			}
	}

	return nil
}

func optimizeResourceDicts(ctx *model.Context) error {
	for i := 1; i <= ctx.PageCount; i++ {
		d, _, inhPAttrs, err := ctx.PageDict(i, true)
		if err != nil {
			return err
		}
		if d == nil {
			continue
		}
		if len(inhPAttrs.Resources) > 0 {
			d["Resources"] = inhPAttrs.Resources
		}
	}
	// TODO Remove resource dicts from inner nodes.
	return nil
}

// OptimizeXRefTable optimizes an xRefTable by locating and getting rid of redundant embedded fonts and images.
func OptimizeXRefTable(ctx *model.Context) error {
	if log.InfoEnabled() {
		log.Info.Println("optimizing fonts & images")
	}
	if log.OptimizeEnabled() {
		log.Optimize.Println("optimizeXRefTable begin")
	}

	// Sometimes free objects are used although they are part of the free object list.
	// Replace references to free xref table entries with a reference to a NULL object.
	if err := fixReferencesToFreeObjects(ctx); err != nil {
		return err
	}

	if ctx.Cmd == model.OPTIMIZE && ctx.Conf.OptimizeResourceDicts {
		// Extra step with potential for performance hit when processing large files.
		if err := optimizeResourceDicts(ctx); err != nil {
			return err
		}
	}

	// Get rid of duplicate embedded fonts and images.
	if err := optimizeFontAndImages(ctx); err != nil {
		return err
	}

	// Get rid of PieceInfo dict from root.
	if err := ctx.DeleteDictEntry(ctx.RootDict, "PieceInfo"); err != nil {
		return err
	}

	// Calculate memory usage of binary content for stats.
	if err := calcBinarySizes(ctx); err != nil {
		return err
	}

	ctx.Optimized = true

	if log.OptimizeEnabled() {
		log.Optimize.Println("optimizeXRefTable end")
	}

	return nil
}
