package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/twitter"
)

func randomString() (string, error) {
	length := 32
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func authMiddleware() gin.HandlerFunc {
	fmt.Println("auth middleware")
	goth.UseProviders(
		twitter.New(
			os.Getenv("TWITTER_KEY"),
			os.Getenv("TWITTER_SECRET"),
			os.Getenv("URL")+"/auth/callback?provider=twitter",
		),
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
	s3.saveUser(username, user.AccessToken+":"+user.AccessTokenSecret)

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
		c.Redirect(http.StatusFound, "/login")
	}
	fmt.Println("username: ", username)

	userData, err := s3.loadUser(username)
	if err != nil {
		fmt.Println("loadUser failed")
		c.Redirect(http.StatusFound, "/login")
	}
	fmt.Println("s3Data:   ", string(userData))

	response, ok := c.GetPostForm("password")
	if !ok {
		fmt.Println("postform failed")
		c.Redirect(http.StatusFound, "/login")
	}
	fmt.Println("response: ", response)

	session := sessions.Default(c)
	challange := session.Get("challange").(string)
	answer := fmt.Sprintf("%x", sha256.Sum256([]byte(string(userData)+challange)))

	fmt.Println("answer:   ", answer)

	if answer == response {
		session.Set("user", username)
		session.Save()
		c.Redirect(http.StatusFound, "/")
		return
	}

	c.Redirect(http.StatusFound, "/login")
}

func getloginfunc(c *gin.Context) {
	challange, err := randomString()
	if err != nil {
		return
	}
	session := sessions.Default(c)
	session.Set("challange", challange)
	session.Save()
	fmt.Println("challange:", challange)
	c.HTML(http.StatusOK, "auth.html", gin.H{
		"Challenge": challange,
	})
}
