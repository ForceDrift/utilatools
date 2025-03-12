package main

import (
	api "example/utilatools/pkg/api/pdf_actions"
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
	{Color: "Blacc", Breed: "Golden (Daquean) Retrievenigga", Age: 2, Food: "KFC", Alcohol: "Hennessy"},
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
	api.RotationRoutes(router)
	router.Run("localhost:8080")
}
