package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
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

func authMiddleware() echo.MiddlewareFunc {
	goth.UseProviders(
		twitter.New(
			os.Getenv("TWITTER_KEY"),
			os.Getenv("TWITTER_SECRET"),
			os.Getenv("URL")+"/auth/callback?provider=twitter",
		),
	)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			cookie, err := c.Cookie("user")
			if err != nil || cookie.Value == "" {
				log.Println("get failed")
				c.Redirect(http.StatusFound, "/login")
			}
			return next(c)
		}
	}
}

func authCallback(c echo.Context) (err error) {
	s3 := c.Get("S3").(*Wikidata)
	user, err := gothic.CompleteUserAuth(c.Response().Writer, c.Request())
	if err != nil {
		log.Println("User auth failed", err)
		c.Redirect(http.StatusFound, "/500")
		return
	}

	var userData userData

	userData.Name = user.Name
	userData.ID = user.Provider + user.UserID
	userData.Token = user.AccessToken
	userData.Secret = user.AccessTokenSecret
	s3.saveUserAsync(user.Name, &userData)

	c.SetCookie(&http.Cookie{Name: "user", Value: userData.ID})

	c.Redirect(http.StatusFound, "/")
	return nil
}

func authenticate(c echo.Context) (err error) {
	gothic.BeginAuthHandler(c.Response().Writer, c.Request())
	return nil
}

func postloginfunc(c echo.Context) (err error) {
	// Forget last cookie first
	c.SetCookie(&http.Cookie{Name: "user"})
	c.SetCookie(&http.Cookie{Name: "breadcrumb"})

	s3 := c.Get("S3").(*Wikidata)

	username := c.FormValue("username")
	if username == "" {
		log.Println("Failed to get username")
		c.Redirect(http.StatusFound, "/login")
		return
	}
	log.Println("username: ", username)

	userData, err := s3.loadUserAsync(username)
	if err != nil {
		log.Println("User is not found")
		c.Redirect(http.StatusFound, "/login")
		return
	}
	log.Println("s3Data:   ", string(userData.Name))

	response := c.FormValue("password")
	log.Println("response: ", response)

	cookie, err := c.Cookie("challange")
	if err != nil {
		return err
	}

	challange := cookie.Value
	// Answer is SHA256(SHA256(password string) + challange string)
	// SHA256(password) should be SHA256(password + salt), but it's too much.
	// Wiki admin or sniffer cannot see raw password string on network and S3.
	// Cookie itself may not safe, if network is http. (Is is encrypted?, but sniffer can see cookie.)
	// Use https proxy, if you want to prevent spoofing.
	answer := fmt.Sprintf("%x", sha256.Sum256([]byte(string(userData.Secret)+challange)))

	log.Println("answer:   ", answer)

	if answer == response {
		c.SetCookie(&http.Cookie{Name: "user", Value: username})
		c.Redirect(http.StatusFound, "/")
		return
	}

	c.Redirect(http.StatusFound, "/login")
	return nil
}

func getloginfunc(c echo.Context) (err error) {
	challange, err := randomString()
	if err != nil {
		return
	}
	c.SetCookie(&http.Cookie{Name: "challange", Value: challange})
	log.Println("challange:", challange)
	c.Render(http.StatusOK, "auth.html", map[string]interface{}{
		"Challenge": challange,
	})
	return nil
}

func getlogoutfunc(c echo.Context) (err error) {
	c.SetCookie(&http.Cookie{Name: "user"})
	c.SetCookie(&http.Cookie{Name: "breadcrumb"})
	c.Redirect(http.StatusFound, "/login")
	return nil
}

func getsignupfunc(c echo.Context) (err error) {
	c.Render(http.StatusOK, "signup.html", map[string]interface{}{})
	return nil
}

func postsignupfunc(c echo.Context) (err error) {
	s3 := c.Get("S3").(*Wikidata)

	var user userData
	user.Name = c.FormValue("username")
	if user.Name == "" {
		log.Println("Failed to get username")
		c.Redirect(http.StatusFound, "/signup")
		return
	}

	_, err = s3.loadUserAsync(user.Name)
	if err == nil {
		log.Println("User already exist: ", user.Name)
		c.Redirect(http.StatusFound, "/signup")
		return
	}

	log.Println("signup: ", user.Name)

	user.Secret = c.FormValue("password")

	err = s3.saveUserAsync(user.Name, &user)
	if err != nil {
		log.Println("saveUser failed", err)
		c.Redirect(http.StatusFound, "/500")
		return
	}

	c.Redirect(http.StatusFound, "/login")
	return nil
}
