package handler

import (
	"net/http"

	"piper_go/internal/svc"
	"piper_go/pkg/auth"
	piperjson "piper_go/pkg/json"
)

func AuthLoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := readBody(r)
		if err != nil {
			writeFailure(w, err)
			return
		}
		cred, err := auth.ParseLoginBody(body)
		if err != nil {
			piperjson.WriteMsg(w, http.StatusOK, piperjson.FailureString("Invalid username or password"))
			return
		}
		token, err := svcCtx.Auth.Login(cred.Username, cred.Password)
		if err != nil {
			piperjson.WriteMsg(w, http.StatusOK, piperjson.FailureString("Invalid username or password"))
			return
		}
		profile, _ := svcCtx.Auth.UserProfile(token)
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(map[string]any{
			"access_token":  token,
			"refresh_token": token,
			"user":          profile,
		}))
	}
}

func AuthLogoutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		svcCtx.Auth.RevokeToken(bearerToken(r))
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(nil))
	}
}

func AuthMeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, err := svcCtx.Auth.UserProfile(bearerToken(r))
		if err != nil {
			piperjson.WriteMsg(w, http.StatusOK, piperjson.FailureString("Not authenticated"))
			return
		}
		piperjson.WriteMsg(w, http.StatusOK, piperjson.Success(profile))
	}
}
