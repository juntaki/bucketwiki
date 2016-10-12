package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/twitter"
)

func authMiddleware() gin.HandlerFunc {
	fmt.Println("auth middleware")
	goth.UseProviders(
		twitter.New(os.Getenv("TWITTER_KEY"), os.Getenv("TWITTER_SECRET"), os.Getenv("URL")+"/auth/callback?provider=twitter"),
	)

	return func(c *gin.Context) {
		session := sessions.Default(c)
		user := session.Get("user")
		if user == nil {
			fmt.Println("get failed")
			c.Redirect(http.StatusFound, "/login")
		}
		c.Next()
	}
}

func authCallback(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	user, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		return
	}
	username := user.Provider + user.UserID
	s3.saveUser(username, user.AccessToken)

	session := sessions.Default(c)
	session.Set("user", username)
	session.Save()

	c.Redirect(http.StatusFound, "/")
}

func authenticate(c *gin.Context) {
	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func postloginfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)
	username, ok := c.GetPostForm("username")
	if !ok {
		fmt.Println("postform failed")
		return
	}
	_, err := s3.loadUser(username)
	if err != nil {
		fmt.Println("loadUser failed")
		return
	}
	session := sessions.Default(c)
	session.Set("user", username)
	session.Save()
	c.Redirect(http.StatusFound, "/")
}

func getloginfunc(c *gin.Context) {
	c.HTML(http.StatusOK, "auth.html", gin.H{})
}
