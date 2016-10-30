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
		c.Redirect(http.StatusInternalServerError, "/500")
		return
	}

	var userData userData

	userData.Name = user.Name
	userData.ID = user.Provider + user.UserID
	userData.Token = user.AccessToken
	userData.Secret = user.AccessTokenSecret
	s3.saveUser(userData)

	session := sessions.Default(c)
	session.Set("user", userData.ID)
	session.Save()

	c.Redirect(http.StatusFound, "/")
}

func authenticate(c *gin.Context) {
	gothic.BeginAuthHandler(c.Writer, c.Request)
}

func postloginfunc(c *gin.Context) {
	// Forget last cookie first
	session := sessions.Default(c)
	session.Delete("user")
	session.Delete("breadcrumb")
	session.Save()

	s3 := c.MustGet("S3").(*Wikidata)

	username, ok := c.GetPostForm("username")
	if !ok || username == "" {
		fmt.Println("Failed to get username")
		c.Redirect(http.StatusFound, "/login")
		return
	}
	fmt.Println("username: ", username)

	userData, err := s3.loadUser(username)
	if err != nil {
		fmt.Println("User is not found")
		c.Redirect(http.StatusFound, "/login")
		return
	}
	fmt.Println("s3Data:   ", string(userData.Name))

	response, ok := c.GetPostForm("password")
	if !ok {
		fmt.Println("Failed to get password")
		c.Redirect(http.StatusFound, "/login")
		return
	}
	fmt.Println("response: ", response)

	challange := session.Get("challange").(string)

	// Answer is SHA256(SHA256(password string) + challange string)
	// SHA256(password) should be SHA256(password + salt), but it's too much.
	// Wiki admin or sniffer cannot see raw password string on network and S3.
	// Cookie itself may not safe, if network is http. (Is is encrypted?, but sniffer can see cookie.)
	// Use https proxy, if you want to prevent spoofing.
	answer := fmt.Sprintf("%x", sha256.Sum256([]byte(string(userData.Secret)+challange)))

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

func getlogoutfunc(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("user")
	session.Delete("breadcrumb")
	session.Save()
	c.Redirect(http.StatusFound, "/login")
}

func getsignupfunc(c *gin.Context) {
	c.HTML(http.StatusOK, "signup.html", gin.H{})
}

func postsignupfunc(c *gin.Context) {
	s3 := c.MustGet("S3").(*Wikidata)

	var user userData
	var ok bool
	user.Name, ok = c.GetPostForm("username")
	if !ok || user.Name == "" {
		fmt.Println("Failed to get username")
		c.Redirect(http.StatusFound, "/signup")
		return
	}

	_, err := s3.loadUser(user.Name)
	if err == nil {
		fmt.Println("User already exist: ", user.Name)
		c.Redirect(http.StatusFound, "/signup")
		return
	}

	fmt.Println("signup: ", user.Name)

	user.Secret, ok = c.GetPostForm("password")
	if !ok {
		fmt.Println("Failed to get password")
		c.Redirect(http.StatusFound, "/signup")
		return
	}

	err = s3.saveUser(user)
	if err != nil {
		fmt.Println("saveUser failed")
		c.Redirect(http.StatusInternalServerError, "/500")
		return
	}

	c.Redirect(http.StatusFound, "/login")
}
