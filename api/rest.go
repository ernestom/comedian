package api

import (
	"fmt"
	"net/http"
	"net/url"

	"strconv"

	"github.com/gorilla/schema"
	"github.com/labstack/echo"
	"github.com/maddevsio/comedian/config"
	"github.com/maddevsio/comedian/model"
	"github.com/maddevsio/comedian/storage"
	log "github.com/sirupsen/logrus"
)

type REST struct {
	db      storage.Storage
	e       *echo.Echo
	c       config.Config
	decoder *schema.Decoder
}

const (
	commandAdd        = "/comedianadd"
	commandRemove     = "/comedianremove"
	commandList       = "/comedianlist"
	commandAddTime    = "/standuptimeset"
	commandRemoveTime = "/standuptimeremove"
	commandTime       = "/standuptime"
)

// NewRESTAPI creates API for Slack commands
func NewRESTAPI(c config.Config) (*REST, error) {
	e := echo.New()
	conn, err := storage.NewMySQL(c)
	if err != nil {
		return nil, err
	}
	r := &REST{
		db:      conn,
		e:       e,
		c:       c,
		decoder: schema.NewDecoder(),
	}
	r.initEndpoints()
	return r, nil
}

func (r *REST) initEndpoints() {
	r.e.POST("/commands", r.handleCommands)
}

// Start starts http server
func (r *REST) Start() error {
	return r.e.Start(r.c.HTTPBindAddr)
}

func (r *REST) handleCommands(c echo.Context) error {
	form, err := c.FormParams()
	if err != nil {
		return c.JSON(http.StatusBadRequest, nil)
	}
	if command := form.Get("command"); command != "" {
		switch command {
		case commandAdd:
			return r.addCommand(c, form)
		case commandRemove:
			return r.removeCommand(c, form)
		case commandList:
			return r.listCommands(c, form)
		case commandAddTime:
			return r.addTime(c, form)
		case commandRemoveTime:
			return r.removeTime(c, form)
		case commandTime:
			return r.listTime(c, form)
		default:
			return c.String(http.StatusNotImplemented, "Not implemented")
		}
	}
	return c.JSON(http.StatusMethodNotAllowed, "Command not allowed")
}

func (r *REST) addCommand(c echo.Context, f url.Values) error {
	var ca FullSlackForm
	if err := r.decoder.Decode(&ca, f); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := ca.Validate(); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	_, err := r.db.CreateChannelIDTextFormUser(model.ChannelIDTextFormUser{
		SlackName: ca.Text,
		ChannelID: ca.ChannelID,
		Channel:   ca.ChannelName,
	})
	if err != nil {
		log.Errorf("could not create standup user: %v", err)
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to create user :%v", err))
	}
	return c.String(http.StatusOK, fmt.Sprintf("%s added", ca.Text))
}

func (r *REST) removeCommand(c echo.Context, f url.Values) error {
	var ca ChannelIDTextForm
	if err := r.decoder.Decode(&ca, f); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := ca.Validate(); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	err := r.db.DeleteChannelIDTextFormUserByUsername(ca.Text, ca.ChannelID)
	if err != nil {
		log.Errorf("could not delete standup: %v", err)
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to delete user :%v", err))
	}
	return c.String(http.StatusOK, fmt.Sprintf("%s deleted", ca.Text))
}

func (r *REST) listCommands(c echo.Context, f url.Values) error {
	var ca ChannelIDForm
	if err := r.decoder.Decode(&ca, f); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := ca.Validate(); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	users, err := r.db.ListChannelIDTextFormUsers(ca.ChannelID)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to list users :%v", err))
	}

	return c.JSON(http.StatusOK, &users)
}
func (r *REST) addTime(c echo.Context, f url.Values) error {

	var ca FullSlackForm
	if err := r.decoder.Decode(&ca, f); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := ca.Validate(); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	timeInt, err := strconv.Atoi(ca.Text)
	if err != nil {
		log.Errorf("could not conver time: %v", err)
	}
	_, err = r.db.CreateChannelIDTextFormTime(model.ChannelIDTextFormTime{
		ChannelID: ca.ChannelID,
		Channel:   ca.ChannelName,
		Time:      int64(timeInt),
	})
	if err != nil {
		log.Errorf("could not create standup time: %v", err)
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to add standup time :%v", err))
	}
	return c.String(http.StatusOK, fmt.Sprintf("standup time at %d added", timeInt))
}

func (r *REST) removeTime(c echo.Context, f url.Values) error {
	var ca ChannelForm
	if err := r.decoder.Decode(&ca, f); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := ca.Validate(); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	err := r.db.DeleteChannelIDTextFormTime(ca.ChannelID)
	if err != nil {
		log.Errorf("could not delete standup time: %v", err)
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to delete standup time :%v", err))
	}
	return c.String(http.StatusOK, fmt.Sprintf("standup time for %s channel deleted", ca.ChannelName))
}

func (r *REST) listTime(c echo.Context, f url.Values) error {
	var ca ChannelIDForm
	if err := r.decoder.Decode(&ca, f); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := ca.Validate(); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	time, err := r.db.ListChannelIDTextFormTime(ca.ChannelID)
	if err != nil {
		log.Println(err)
		return c.String(http.StatusBadRequest, fmt.Sprintf("failed to list time :%v", err))
	}

	return c.JSON(http.StatusOK, &time.Time)
}