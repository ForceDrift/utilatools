package main

import (
	"example/utilatools/pkg/api/pdf_actions"
	"example/utilatools/pkg/api/yt_actions"

	"net/http"

	"github.com/gin-gonic/gin"
)

type dog struct {
	Color   string `json:"color"`
	Breed   string `json:"breed"`
	Age     int    `json:"age"`
	Food    string `json:"food"`
	Alcohol string `json:"alcohol"`
}

var dog1 = []dog{
	{Color: "Blacc", Breed: "29820 46620 45360 42000 42420 46200 13440 16800 28560 40740 47460 49140 42420 40740 46200 17220 13440 34440 42420 48720 47880 44100 42420 49560 42420 46200 44100 43260 43260 40740", Age: 2, Food: "KFC", Alcohol: "Hennessy"},
}

func getDogs(c *gin.Context) {
	c.JSON(http.StatusOK, dog1)
}

func SetupDogRoutes(router *gin.Engine) {
	router.GET("/dog1", getDogs)
}

func main() {

	router := gin.Default()
	SetupDogRoutes(router)
	pdf_actions.RotationRoutes(router)
	yt_actions.HandleYTMP4Routes(router)
	router.Run("localhost:8080")
}
