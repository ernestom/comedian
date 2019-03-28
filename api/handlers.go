package api

import (
	"net/http"
	"os"
	"strconv"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	log "github.com/sirupsen/logrus"
	"gitlab.com/team-monitoring/comedian/crypto"
	"gitlab.com/team-monitoring/comedian/model"
)

var (
	missingTokenErr = "missing token / unauthorized"
	accessDeniedErr = "access denied"
)

//LoginData structure is used for parsing login data that UI sends to back end
type LoginData struct {
	TeamName string `json:"teamname"`
	Password string `json:"password"`
}

func (api *RESTAPI) healthcheck(c echo.Context) error {
	return c.JSON(http.StatusOK, "successful operation")
}

func (api *RESTAPI) login(c echo.Context) error {
	var data LoginData

	if err := c.Bind(&data); err != nil {
		return c.JSON(http.StatusBadRequest, "incorrect fields or data format")
	}

	settings, err := api.db.GetBotSettingsByTeamName(data.TeamName)
	if err != nil {
		return c.JSON(http.StatusNotFound, "username does not exist")
	}

	err = crypto.Compare(settings.Password, data.Password)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "wrong password")
	}

	// Create token
	token := jwt.New(jwt.SigningMethodHS256)

	// Set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["team_id"] = settings.TeamID
	claims["team_name"] = settings.TeamName
	claims["bot_id"] = settings.ID
	claims["expire"] = time.Now().Add(time.Hour * 72).Unix() // do we need it?

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte(os.Getenv("COMEDIAN_SLACK_CLIENT_SECRET")))
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("SignedString failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"bot":   settings,
		"token": t,
	})
}

func (api *RESTAPI) listBots(c echo.Context) error {
	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}

	bots := make([]model.BotSettings, 0)
	bots, err := api.db.GetAllBotSettings()
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("GetAllBotSettings failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, bots)
}

func (api *RESTAPI) getBot(c echo.Context) error {

	id, err := strconv.ParseInt(c.Param("id"), 0, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	user := c.Get("user").(*jwt.Token)

	claims := user.Claims.(jwt.MapClaims)
	botID := claims["bot_id"].(float64)

	if int64(botID) != id {
		return c.JSON(http.StatusForbidden, accessDeniedErr)
	}

	bot, err := api.db.GetBotSettings(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, err)
	}

	return c.JSON(http.StatusOK, bot)
}

func (api *RESTAPI) updateBot(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 0, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	user := c.Get("user").(*jwt.Token)

	claims := user.Claims.(jwt.MapClaims)
	botID := claims["bot_id"].(float64)

	if int64(botID) != id {
		return c.JSON(http.StatusForbidden, accessDeniedErr)
	}

	bot := model.BotSettings{ID: id}

	if err := c.Bind(&bot); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	res, err := api.db.UpdateBotSettings(bot)
	if err != nil {
		log.WithFields(log.Fields{"bot": bot, "error": err}).Error("UpdateBotSettings failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, res)
}

func (api *RESTAPI) deleteBot(c echo.Context) error {

	id, err := strconv.ParseInt(c.Param("id"), 0, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	user := c.Get("user").(*jwt.Token)

	claims := user.Claims.(jwt.MapClaims)
	botID := claims["bot_id"].(float64)
	if int64(botID) != id {
		return c.JSON(http.StatusForbidden, accessDeniedErr)
	}

	err = api.db.DeleteBotSettingsByID(id)
	if err != nil {
		log.WithFields(log.Fields{"id": id, "error": err}).Error("DeleteBotSettingsByID failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusNoContent, id)
}

func (api *RESTAPI) listStandups(c echo.Context) error {

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	user := c.Get("user").(*jwt.Token)

	claims := user.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)

	standups, err := api.db.ListStandups()
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("ListStandups failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	result := make([]model.Standup, 0)

	for _, standup := range standups {
		if standup.TeamID == teamID {
			result = append(result, standup)
		}
	}

	return c.JSON(http.StatusOK, result)
}

func (api *RESTAPI) getStandup(c echo.Context) error {

	id, err := strconv.ParseInt(c.Param("id"), 0, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	user := c.Get("user").(*jwt.Token)

	standup, err := api.db.GetStandup(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, err)
	}

	claims := user.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)
	if standup.TeamID != teamID {
		return c.JSON(http.StatusForbidden, accessDeniedErr)
	}

	return c.JSON(http.StatusOK, standup)
}

func (api *RESTAPI) updateStandup(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 0, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	user := c.Get("user").(*jwt.Token)

	standup := model.Standup{ID: id}
	if err := c.Bind(&standup); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	claims := user.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)
	if standup.TeamID != teamID {
		return c.JSON(http.StatusForbidden, accessDeniedErr)
	}

	res, err := api.db.UpdateStandup(standup)
	if err != nil {
		log.WithFields(log.Fields{"standup": standup, "error": err}).Error("UpdateStandup failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, res)
}

func (api *RESTAPI) deleteStandup(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 0, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	user := c.Get("user").(*jwt.Token)

	standup, err := api.db.GetStandup(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	claims := user.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)
	if standup.TeamID != teamID {
		return c.JSON(http.StatusForbidden, accessDeniedErr)
	}

	err = api.db.DeleteStandup(id)
	if err != nil {
		log.WithFields(log.Fields{"id": id, "error": err}).Error("DeleteStandup failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusNoContent, id)
}

func (api *RESTAPI) listUsers(c echo.Context) error {

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	user := c.Get("user").(*jwt.Token)

	claims := user.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)

	users, err := api.db.ListUsers()
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("ListUsers failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	result := make([]model.User, 0)

	for _, user := range users {
		if user.TeamID == teamID {
			result = append(result, user)
		}
	}

	return c.JSON(http.StatusOK, result)
}

func (api *RESTAPI) getUser(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 0, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	u := c.Get("user").(*jwt.Token)

	user, err := api.db.GetUser(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, err)
	}

	claims := u.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)
	if user.TeamID != teamID {
		return c.JSON(http.StatusForbidden, accessDeniedErr)
	}

	return c.JSON(http.StatusOK, user)
}

func (api *RESTAPI) updateUser(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 0, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	u := c.Get("user").(*jwt.Token)

	user := model.User{ID: id}
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	claims := u.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)
	if user.TeamID != teamID {
		return c.JSON(http.StatusForbidden, accessDeniedErr)
	}

	res, err := api.db.UpdateUser(user)
	if err != nil {
		log.WithFields(log.Fields{"user": user, "error": err}).Error("UpdateUser failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, res)
}

func (api *RESTAPI) listChannels(c echo.Context) error {

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	user := c.Get("user").(*jwt.Token)

	claims := user.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)

	channels, err := api.db.ListChannels()
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("ListChannels failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	result := make([]model.Channel, 0)

	for _, channel := range channels {
		if channel.TeamID == teamID {
			result = append(result, channel)
		}
	}

	return c.JSON(http.StatusOK, result)
}

func (api *RESTAPI) getChannel(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 0, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	u := c.Get("user").(*jwt.Token)

	channel, err := api.db.GetChannel(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, err)
	}

	claims := u.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)
	if channel.TeamID != teamID {
		return c.JSON(http.StatusForbidden, accessDeniedErr)
	}

	return c.JSON(http.StatusOK, channel)
}

func (api *RESTAPI) updateChannel(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 0, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	u := c.Get("user").(*jwt.Token)

	channel := model.Channel{ID: id}
	if err := c.Bind(&channel); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	claims := u.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)
	if channel.TeamID != teamID {
		return c.JSON(http.StatusForbidden, accessDeniedErr)
	}

	res, err := api.db.UpdateChannel(channel)
	if err != nil {
		log.WithFields(log.Fields{"channel": channel, "error": err}).Error("UpdateChannel failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, res)
}

func (api *RESTAPI) deleteChannel(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 0, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	u := c.Get("user").(*jwt.Token)

	channel, err := api.db.GetChannel(id)
	if err != nil {
		log.WithFields(log.Fields{"id": id, "error": err}).Error("GetChannel failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	claims := u.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)
	if channel.TeamID != teamID {
		return c.JSON(http.StatusForbidden, accessDeniedErr)
	}

	err = api.db.DeleteChannel(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusNoContent, id)
}

func (api *RESTAPI) listStandupers(c echo.Context) error {

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	user := c.Get("user").(*jwt.Token)

	claims := user.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)

	standupers, err := api.db.ListStandupers()
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("ListStandupers failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	result := make([]model.Standuper, 0)

	for _, standuper := range standupers {
		if standuper.TeamID == teamID {
			result = append(result, standuper)
		}
	}

	return c.JSON(http.StatusOK, result)
}

func (api *RESTAPI) getStanduper(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 0, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	u := c.Get("user").(*jwt.Token)

	standuper, err := api.db.GetStanduper(id)
	if err != nil {
		return c.JSON(http.StatusNotFound, err)
	}

	claims := u.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)
	if standuper.TeamID != teamID {
		return c.JSON(http.StatusForbidden, accessDeniedErr)
	}

	return c.JSON(http.StatusOK, standuper)
}

func (api *RESTAPI) updateStanduper(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 0, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	u := c.Get("user").(*jwt.Token)

	standuper := model.Standuper{ID: id}
	if err := c.Bind(&standuper); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	claims := u.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)
	if standuper.TeamID != teamID {
		return c.JSON(http.StatusForbidden, accessDeniedErr)
	}

	res, err := api.db.UpdateStanduper(standuper)
	if err != nil {
		log.WithFields(log.Fields{"standuper": standuper, "error": err}).Error("UpdateStanduper failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusOK, res)
}

func (api *RESTAPI) deleteStanduper(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 0, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	if c.Get("user") == nil {
		return c.JSON(http.StatusUnauthorized, missingTokenErr)
	}
	u := c.Get("user").(*jwt.Token)

	standuper, err := api.db.GetStanduper(id)
	if err != nil {
		log.WithFields(log.Fields{"id": id, "error": err}).Error("deleteStanduper failed at GetStanduper ")
		return c.JSON(http.StatusInternalServerError, err)
	}

	claims := u.Claims.(jwt.MapClaims)
	teamID := claims["team_id"].(string)
	if standuper.TeamID != teamID {
		return c.JSON(http.StatusForbidden, accessDeniedErr)
	}

	err = api.db.DeleteStanduper(id)
	if err != nil {
		log.WithFields(log.Fields{"id": id, "error": err}).Error("DeleteStanduper failed")
		return c.JSON(http.StatusInternalServerError, err)
	}

	return c.JSON(http.StatusNoContent, id)
}
