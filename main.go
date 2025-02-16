package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Error struct {
	Code     string
	Subtitle string
	Details  string
}

func (e *Error) MarshalJSON() ([]byte, error) {
	type errorPartitial Error
	type errorComplete struct {
		Status string
		Type   string
		Title  string
		Image  string
		*errorPartitial
	}
	return json.Marshal(errorComplete{
		Status:         "error",
		Type:           "error",
		Title:          "ERROR",
		Image:          "https://cdn.discordapp.com/emojis/867801176565481472.webp",
		errorPartitial: (*errorPartitial)(e),
	})
}

func main() {
	LoadConfig()
	LoadFiles()

	e := echo.New()

	e.Use(middleware.Logger())

	e.GET("/api/capabilities", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"api/companion/list": "v1",
			"api/woka/list":      "v1",
		})
	})

	e.GET("/api/map", func(c echo.Context) error {
		var Request struct {
			PlayUri     string `query:"playUri"`
			UserId      string `query:"userId"`
			AccessToken string `query:"accessToken"`
		}
		if err := c.Bind(&Request); err != nil {
			return c.JSON(400, &Error{
				Code:     "INVALID_REQUEST",
				Subtitle: "Failed to bind request",
				Details:  err.Error(),
			})
		}
		playUri, err := url.Parse(Request.PlayUri)
		if err != nil {
			return c.JSON(400, &Error{
				Code:     "INVALID_REQUEST",
				Subtitle: "Invalid playUri",
				Details:  fmt.Sprintf("playUri is not valid url: %s", Request.PlayUri),
			})
		}
		mapId := playUri.Path

		if redirectTo, ok := Config.Redirects[mapId]; ok {
			type redirectError struct {
				RedirectUrl string `json:"redirectUrl"`
			}
			return c.JSON(200, redirectError{
				RedirectUrl: redirectTo,
			})
		}

		m, ok := Config.Maps[mapId]
		if !ok {
			return c.JSON(404, &Error{
				Code:     "UNKNOWN_ROOM",
				Subtitle: "Unknown room",
				Details:  "Failed to find matching room: " + mapId,
			})
		}

		if Request.UserId != "" || Request.AccessToken != "" {
			user, err := GetUserinfo(Request.UserId, Request.AccessToken)
			if err != nil {
				code := 500
				if err, ok := err.(*HttpError); ok {
					code = err.StatusCode
				}
				return c.JSON(code, &Error{
					Code:     "FAILED_TO_GET_USERINFO",
					Subtitle: "Failed to get userinfo",
					Details:  err.Error(),
				})
			}

			_, allowed, _ := GetUserPermission(&m, user.Sub)
			if !allowed {
				return c.JSON(403, &Error{
					Code:     "ACCESS_DENIED",
					Subtitle: "You are not allowed to access this room",
					Details:  "You are not in allowed_tags list",
				})
			}
		}

		type MapResp struct {
			MapStruct
			AuthenticationMandatory bool   `json:"authenticationMandatory"`
			OpidWokaNamePolicy      string `json:"opidWokaNamePolicy"`
		}
		return c.JSON(200, MapResp{
			MapStruct:               m,
			AuthenticationMandatory: true,
			OpidWokaNamePolicy:      "allow_override_opid",
		})
	}, requireAuth)
	e.GET("/api/room/access", func(c echo.Context) error {
		var Request struct {
			UserIdentifier      string   `query:"userIdentifier"`
			IsLogged            string   `query:"isLogged"`
			AccessToken         string   `query:"accessToken"`
			PlayUri             string   `query:"playUri"`
			CharacterTextureIds []string `query:"characterTextureIds[]"`
			CompanionTextureId  string   `query:"companionTextureId"`
		}
		if err := c.Bind(&Request); err != nil {
			return c.JSON(400, &Error{
				Code:     "INVALID_REQUEST",
				Subtitle: "Failed to bind request",
				Details:  err.Error(),
			})
		}
		playUri, err := url.Parse(Request.PlayUri)
		if err != nil {
			return c.JSON(400, &Error{
				Code:     "INVALID_REQUEST",
				Subtitle: "Invalid playUri",
				Details:  fmt.Sprintf("playUri is not valid url: %s", Request.PlayUri),
			})
		}
		mapId := playUri.Path
		m, ok := Config.Maps[mapId]
		if !ok {
			return c.JSON(404, &Error{
				Code:     "UNKNOWN_ROOM",
				Subtitle: "Unknown room",
				Details:  "Failed to find matching room: " + mapId,
			})
		}
		user, err := GetUserinfo(Request.UserIdentifier, Request.AccessToken)
		if err != nil {
			code := 500
			if err, ok := err.(*HttpError); ok {
				code = err.StatusCode
			}
			return c.JSON(code, &Error{
				Code:     "FAILED_TO_GET_USERINFO",
				Subtitle: "Failed to get userinfo",
				Details:  err.Error(),
			})
		}
		type CharacterTexture struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		}

		type CompanionTexture struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		}

		type MemberResp struct {
			Status                   string             `json:"status"`
			Email                    string             `json:"email"`
			Username                 string             `json:"username"`
			UUID                     string             `json:"userUuid"`
			Tags                     []string           `json:"tags"`
			VisitCardUrl             string             `json:"visitCardUrl"`
			IsCharacterTexturesValid bool               `json:"isCharacterTexturesValid"`
			CharacterTextures        []CharacterTexture `json:"characterTextures"`
			IsCompanionTextureValid  bool               `json:"isCompanionTextureValid"`
			CompanionTexture         *CompanionTexture  `json:"companionTexture,omitempty"`
			Messages                 []string           `json:"messages"`
			CanEdit                  bool               `json:"canEdit"`
			World                    string             `json:"world"`
		}

		tags, allowed, editable := GetUserPermission(&m, user.Sub)

		if !allowed {
			return c.JSON(403, &Error{
				Code:     "ACCESS_DENIED",
				Subtitle: "You are not allowed to access this room",
				Details:  "You are not in allowed_tags list",
			})
		}

		resp := MemberResp{
			Status:                   "ok",
			Email:                    user.Email,
			Username:                 user.PrefferedUsername,
			UUID:                     uuid.NewSHA1(Config.UUIDSpace, []byte(user.Sub)).String(),
			Tags:                     tags,
			VisitCardUrl:             "https://example.com",
			IsCharacterTexturesValid: true,
			CharacterTextures:        make([]CharacterTexture, len(Request.CharacterTextureIds)),
			IsCompanionTextureValid:  true,
			CompanionTexture:         nil,
			Messages:                 []string{"welcome"},
			CanEdit:                  editable,
			World:                    m.Group,
		}
		for i, v := range Request.CharacterTextureIds {
			url, ok := WokaKV[v]
			if !ok {
				resp.IsCharacterTexturesValid = false
			}
			resp.CharacterTextures[i] = CharacterTexture{
				ID:  v,
				URL: url,
			}
		}
		if Request.CompanionTextureId != "" {
			url, ok := CompanionKV[Request.CompanionTextureId]
			if !ok {
				resp.IsCompanionTextureValid = false
			}
			resp.CompanionTexture = &CompanionTexture{
				ID:  Request.CompanionTextureId,
				URL: url,
			}
		}
		return c.JSON(200, resp)
	}, requireAuth)

	e.GET("/api/room/sameWorld", func(c echo.Context) error {
		var Request struct {
			RoomUrl string `query:"roomUrl"`
		}
		if err := c.Bind(&Request); err != nil {
			return c.JSON(400, &Error{
				Code:     "INVALID_REQUEST",
				Subtitle: "Failed to bind request",
				Details:  err.Error(),
			})
		}
		playUri, err := url.Parse(Request.RoomUrl)
		if err != nil {
			return c.JSON(400, &Error{
				Code:     "INVALID_REQUEST",
				Subtitle: "Invalid roomUrl",
				Details:  fmt.Sprintf("roomUrl is not valid url: %s", Request.RoomUrl),
			})
		}
		mapId := playUri.Path
		m, ok := Config.Maps[mapId]
		if !ok {
			return c.JSON(404, &Error{
				Code:     "UNKNOWN_ROOM",
				Subtitle: "Unknown room",
				Details:  "Failed to find matching room: " + mapId,
			})
		}
		type Room struct {
			RoomUrl string `json:"roomUrl"`
			WamUrl  string `json:"wamUrl"`
			Name    string `json:"name"`
		}
		resp := make([]Room, 0, len(GroupMap[m.Group]))
		for _, roomId := range GroupMap[m.Group] {
			resp = append(resp, Room{
				RoomUrl: roomId,
				WamUrl:  Config.Maps[roomId].WamUrl,
				Name:    Config.Maps[roomId].RoomName,
			})
		}
		return c.JSON(200, resp)
	}, requireAuth)

	e.GET("/api/companion/list", func(c echo.Context) error {
		return c.JSON(200, CompanionDB)
	}, requireAuth)
	e.GET("/api/woka/list", func(c echo.Context) error {
		return c.JSON(200, WokaDB)
	}, requireAuth)

	log.Print("Starting server")

	log.Print("Server end: ", e.Start(":8080"))
}

type Userinfo struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	PrefferedUsername string `json:"preffered_username"`
}

type HttpError struct {
	StatusCode int
}

func (e *HttpError) Error() string {
	return fmt.Sprint("request failed: ", e.StatusCode)
}

func GetUserinfo(username string, token string) (*Userinfo, error) {
	filename := "users/" + username + ".json"

	fetched, fetchErr := fetchUserinfo(token)
	if fetchErr == nil {
		if username != "" && fetched.Sub != username {
			return nil, errors.New("sub not match")
		}
		// save
		f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if err := json.NewEncoder(f).Encode(fetched); err != nil {
			return nil, err
		}
		return fetched, nil
	}

	// load
	f, err := os.Open(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fetchErr
		}
		return nil, err
	}
	defer f.Close()
	user := Userinfo{}
	if err := json.NewDecoder(f).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func fetchUserinfo(token string) (*Userinfo, error) {
	if token == "" {
		return nil, errors.New("empty token")
	}
	req, err := http.NewRequest("GET", Config.UserinfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, &HttpError{
			StatusCode: resp.StatusCode,
		}
	}
	var user Userinfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserPermission(m *MapStruct, userId string) (tags []string, access bool, edit bool) {
	tags, ok := Config.Tags[userId]
	if !ok {
		tags = []string{"everyone", "default"}
	}

	if len(m.EditorTags) == 0 {
		edit = true
	} else {
		for _, v := range m.EditorTags {
			if edit {
				break
			}
			for _, tag := range tags {
				if tag == v {
					edit = true
					access = true
					break
				}
			}
		}
	}

	if len(m.AllowedTags) == 0 {
		access = true
	} else {
		for _, v := range m.AllowedTags {
			if access {
				break
			}
			for _, tag := range tags {
				if tag == v {
					access = true
					break
				}
			}
		}
	}

	return
}

func requireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().Header.Get("Authorization") != Config.Token {
			return c.JSON(403, &Error{
				Code:     "INVALID_ADMIN_TOKEN",
				Subtitle: "Failed to authenticate token",
				Details:  "authentication bearer token did not match",
			})
		}
		return next(c)
	}
}
