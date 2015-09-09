package server

import (
	"time"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/ungerik/go-gravatar"

	log "github.com/drone/drone/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	common "github.com/drone/drone/pkg/types"
)

// GetLogin accepts a request to authorize the user and to
// return a valid OAuth2 access token. The access token is
// returned as url segment #access_token
//
//     GET /authorize
//
func GetLogin(c *gin.Context) {
	session := ToSession(c)
	remote := ToRemote(c)
	store := ToDatastore(c)

	// when dealing with redirects we may need
	// to adjust the content type. I cannot, however,
	// rememver why, so need to revisit this line.
	c.Writer.Header().Del("Content-Type")

	// TODO: move this back to the remote section
	getLoginOauth2(c)

	// exit if authorization fails
	if c.Writer.Status() != 200 {
		return
	}

	login := ToUser(c)

	// check organization membership, if applicable
	if len(remote.GetOrgs()) != 0 {
		orgs, _ := remote.Orgs(login)
		if !checkMembership(orgs, remote.GetOrgs()) {
			c.Redirect(303, "/login#error=access_denied_org")
			return
		}
	}

	// get the user from the database
	u, err := store.UserLogin(login.Login)
	if err != nil {
		count, err := store.UserCount()
		if err != nil {
			log.Errorf("cannot register %s. %s", login.Login, err)
			c.Redirect(303, "/login#error=internal_error")
			return
		}

		// if self-registration is disabled we should
		// return a notAuthorized error. the only exception
		// is if no users exist yet in the system we'll proceed.
		if !remote.GetOpen() && count != 0 {
			log.Errorf("cannot register %s. registration closed", login.Login)
			c.Redirect(303, "/login#error=access_denied")
			return
		}

		// create the user account
		u = &common.User{}
		u.Login = login.Login
		u.Token = login.Token
		u.Secret = login.Secret
		u.Email = login.Email
		u.Avatar = login.Avatar
		u.Hash = common.GenerateToken()

		// insert the user into the database
		if err := store.AddUser(u); err != nil {
			log.Errorf("cannot insert %s. %s", login.Login, err)
			c.Redirect(303, "/login#error=internal_error")
			return
		}

		// if this is the first user, they
		// should be an admin.
		if count == 0 {
			u.Admin = true
		}
	}

	// update the user meta data and authorization
	// data and cache in the datastore.
	u.Token = login.Token
	u.Secret = login.Secret
	u.Email = login.Email
	u.Avatar = login.Avatar

	// TODO: remove this once gitlab implements setting
	// avatar in the remote package, similar to github
	if len(u.Avatar) == 0 {
		u.Avatar = gravatar.Hash(u.Email)
	}

	if err := store.SetUser(u); err != nil {
		log.Errorf("cannot update %s. %s", u.Login, err)
		c.Redirect(303, "/login#error=internal_error")
		return
	}

	token := &common.Token{
		Kind:   common.TokenSess,
		Login:  u.Login,
		Issued: time.Now().UTC().Unix(),
	}
	tokenstr, err := session.GenerateToken(token)
	if err != nil {
		log.Errorf("cannot create token for %s. %s", u.Login, err)
		c.Redirect(303, "/login#error=internal_error")
		return
	}
	c.Redirect(303, "/#access_token="+tokenstr)
}

// getLoginOauth2 is the default authorization implementation
// using the oauth2 protocol.
func getLoginOauth2(c *gin.Context) {
	var remote = ToRemote(c)

	// Bugagazavr: I think this must be moved to remote config
	//var scope = strings.Join(settings.Auth.Scope, ",")
	//if scope == "" {
	//	scope = remote.Scope()
	//}
	var transport = remote.Oauth2Transport(c.Request)

	// get the OAuth code
	var code = c.Request.FormValue("code")
	//var state = c.Request.FormValue("state")
	if len(code) == 0 {
		// TODO this should be a random number, verified by a cookie
		c.Redirect(303, transport.AuthCodeURL("random"))
		return
	}

	// exhange for a token
	var token, err = transport.Exchange(code)
	if err != nil {
		log.Errorf("cannot get access_token. %s", err)
		c.Redirect(303, "/login#error=token_exchange")
		return
	}

	// get user account
	user, err := remote.Login(token.AccessToken, token.RefreshToken)
	if err != nil {
		log.Errorf("cannot get user with access_token. %s", err)
		c.Redirect(303, "/login#error=user_not_found")
		return
	}

	// add the user to the request
	c.Set("user", user)
}

// getLoginOauth1 is able to authorize a user with the oauth1
// authentication protocol. This is used primarily with Bitbucket
// and Stash only, and one day I hope can be removed.
func getLoginOauth1(c *gin.Context) {

}

// getLoginBasic is able to authorize a user with a username and
// password. This can be used for systems that do not support oauth.
func getLoginBasic(c *gin.Context) {
	var (
		remote   = ToRemote(c)
		username = c.Request.FormValue("username")
		password = c.Request.FormValue("password")
	)

	// get user account
	user, err := remote.Login(username, password)
	if err != nil {
		log.Errorf("invalid username or password for %s. %s", username, err)
		c.Redirect(303, "/login#error=invalid_credentials")
		return
	}

	// add the user to the request
	c.Set("user", user)
}

// checkMembership is a helper function that compares the user's
// organization list to a whitelist of organizations that are
// approved to use the system.
func checkMembership(orgs, whitelist []string) bool {
	orgs_ := make(map[string]struct{}, len(orgs))
	for _, org := range orgs {
		orgs_[org] = struct{}{}
	}
	for _, org := range whitelist {
		if _, ok := orgs_[org]; ok {
			return true
		}
	}
	return false
}
