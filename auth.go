package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/labstack/echo"
	"github.com/markbates/goth/gothic"
	"github.com/pkg/errors"
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

func (h *handler) getSession(c echo.Context) (sess *sessionData, err error) {
	sessionID, err := c.Cookie("sessionID")
	if err != nil {
		return nil, errors.Wrap(err, "cookie not found")
	}
	log.Println("sessionID", sessionID.Value)
	sess = &sessionData{ID: sessionID.Value}
	err = h.db.loadBare(sess)
	if err != nil {
		return nil, errors.Wrap(err, "loadBare failed")
	}
	return sess, nil
}

func (h *handler) setSession(c echo.Context, sess *sessionData) (err error) {
	sess.ID, err = randomString()
	if err != nil {
		return err
	}
	log.Println("save session", sess.ID)
	err = h.db.saveBare(sess)
	if err != nil {
		return err
	}
	c.SetCookie(&http.Cookie{Name: "sessionID", Value: sess.ID, Path: "/"})
	return nil
}

func (h *handler) authMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			log.Println("Auth middleware")
			session, err := h.getSession(c)
			if err != nil {
				log.Println("get session failed", err)
				return c.Redirect(http.StatusFound, "/login")
			}
			if session.Login == false {
				log.Println("not login session")
				return c.Redirect(http.StatusFound, "/login")
			}
			c.Set("session", session)
			return next(c)
		}
	}
}

func (h *handler) authCallbackHandler(c echo.Context) (err error) {
	user, err := gothic.CompleteUserAuth(c.Response().Writer, c.Request())
	if err != nil {
		log.Println("User auth failed", err)
		return c.Redirect(http.StatusFound, "/500")
	}

	var userData userData

	userData.Name = user.Name
	userData.ID = user.Provider + user.UserID
	userData.Token = user.AccessToken
	userData.Secret = user.AccessTokenSecret
	err = h.db.saveBare(&userData)
	if err != nil {
		return err
	}

	sess := &sessionData{
		Login: true,
	}
	err = h.setSession(c, sess)
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, "/")
}

func (h *handler) authHandler(c echo.Context) (err error) {
	url, err := gothic.GetAuthURL(c.Response().Writer, c.Request())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	c.Redirect(http.StatusTemporaryRedirect, url)
	return nil
}

func (h *handler) loginHandler(c echo.Context) (err error) {
	username := c.FormValue("username")
	if username == "" {
		log.Println("Failed to get username")
		return c.Redirect(http.StatusFound, "/login")
	}
	log.Println("username: ", username)
	userData := &userData{
		Name: username,
	}
	err = h.db.loadBare(userData)
	if err != nil {
		log.Println("User is not found", err)
		return c.Redirect(http.StatusFound, "/login")
	}
	log.Println("userData.Name:", userData.Name)

	response := c.FormValue("password")
	log.Println("response: ", response)

	sess, err := h.getSession(c)
	if err != nil {
		return err
	}

	challange := sess.Challange
	// Answer is SHA256(SHA256(password string) + challange string)
	// SHA256(password) should be SHA256(password + salt), but it's too much.
	// Wiki admin or sniffer cannot see raw password string on network and S3.
	answer := fmt.Sprintf("%x", sha256.Sum256([]byte(string(userData.Secret)+challange)))

	log.Println("answer:   ", answer)

	if answer == response {
		log.Println("good answer")
		sess.Login = true
		sess.User = username
		err := h.setSession(c, sess)
		if err != nil {
			return err
		}
		return c.Redirect(http.StatusFound, "/")
	}
	log.Println("bad answer")
	return c.Redirect(http.StatusFound, "/login")
}

func (h *handler) loginPageHandler(c echo.Context) (err error) {
	challange, err := randomString()
	if err != nil {
		return nil
	}

	sess := &sessionData{
		Login:     false,
		Challange: challange,
	}
	err = h.setSession(c, sess)
	if err != nil {
		return err
	}

	log.Println("challange:", challange)
	return c.Render(http.StatusOK, "auth.html", map[string]interface{}{
		"Challenge": challange,
	})
}

func (h *handler) logoutHandler(c echo.Context) (err error) {
	sess := c.Get("session").(*sessionData)
	err = h.db.deleteBare(&sessionData{ID: sess.ID})
	if err != nil {
		return err
	}
	return c.Redirect(http.StatusFound, "/login")
}

func (h *handler) signupPageHandler(c echo.Context) (err error) {
	return c.Render(http.StatusOK, "signup.html", map[string]interface{}{})
}

func (h *handler) signupHandler(c echo.Context) (err error) {
	var user userData
	user.Name = c.FormValue("username")
	if user.Name == "" {
		log.Println("Failed to get username")
		return c.Redirect(http.StatusFound, "/signup")
	}

	err = h.db.loadBare(&user)
	if err == nil {
		log.Println("User already exist: ", user.Name)
		return c.Redirect(http.StatusFound, "/signup")
	}

	log.Println("signup: ", user.Name)

	user.Secret = c.FormValue("password")

	err = h.db.saveBare(&user)
	if err != nil {
		log.Println("saveUser failed", err)
		return c.Redirect(http.StatusFound, "/500")
	}

	return c.Redirect(http.StatusFound, "/login")
}
