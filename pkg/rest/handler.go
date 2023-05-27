package rest

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/sunyakun/gearbox/pkg/apis"
	pkgerrors "github.com/sunyakun/gearbox/pkg/errors"
	"github.com/emicklei/go-restful/v3"
	"github.com/sirupsen/logrus"
)

type Handler[T any, PT interface {
	apis.Object
	*T
}] struct {
	resource     Resource[PT]
	resourceName string
}

func NewHandler[T any, PT interface {
	apis.Object
	*T
}](resource Resource[PT]) *Handler[T, PT] {
	return &Handler[T, PT]{
		resource:     resource,
		resourceName: resource.Name(),
	}
}

func (hdl *Handler[T, PT]) Error(req *restful.Request, resp *restful.Response, err error) {
	var status apis.Status
	if apiStatus, ok := err.(pkgerrors.APIStatus); ok {
		status = apiStatus.Status()
	} else {
		status = pkgerrors.NewInternalError(err).Status()
	}

	resp.WriteHeader(status.Code)

	err = resp.WriteEntity(status)
	if err != nil {
		logrus.Errorf("internal server error: %s", err)
		resp.InternalServerError()
	}
}

func (hdl *Handler[T, PT]) Get(req *restful.Request, resp *restful.Response) {
	resource, err := hdl.resource.Get(req.Request.Context(), req.PathParameter(hdl.resourceName))
	if err != nil {
		hdl.Error(req, resp, err)
		return
	}
	if err := resp.WriteAsJson(resource); err != nil {
		hdl.Error(req, resp, err)
		return
	}
}

func (hdl *Handler[T, PT]) List(req *restful.Request, resp *restful.Response) {
	offset := req.QueryParameter("offset")
	limit := req.QueryParameter("limit")

	var (
		err       error
		offsetVal int
		limitVal  int
	)

	if offset == "" {
		offsetVal = 0
	} else {
		offsetVal, err = strconv.Atoi(offset)
		if err != nil {
			hdl.Error(req, resp, err)
			return
		}
	}

	if limit == "" {
		limitVal = 10
	} else {
		limitVal, err = strconv.Atoi(limit)
		if err != nil {
			hdl.Error(req, resp, err)
			return
		}
	}

	objs, cnt, err := hdl.resource.GetList(req.Request.Context(), apis.ListOptions{
		Offset:   offsetVal,
		Limit:    limitVal,
		Selector: req.QueryParameter("selector"),
	})
	if err != nil {
		hdl.Error(req, resp, err)
		return
	}

	objList := apis.ObjectList[PT]{
		Count: cnt,
		Items: objs,
	}

	if int64(offsetVal)+int64(limitVal) < cnt {
		objList.Continue = true
	}

	err = resp.WriteAsJson(objList)
	if err != nil {
		hdl.Error(req, resp, err)
		return
	}
}

func (hdl *Handler[T, PT]) Update(req *restful.Request, resp *restful.Response) {
	var t PT = new(T)
	if err := req.ReadEntity(t); err != nil {
		hdl.Error(req, resp, err)
		return
	}
	if err := hdl.resource.Update(req.Request.Context(), req.PathParameter(hdl.resourceName), t); err != nil {
		hdl.Error(req, resp, err)
		return
	}
	if err := resp.WriteAsJson(t); err != nil {
		hdl.Error(req, resp, err)
		return
	}
}

func (hdl *Handler[T, PT]) Delete(req *restful.Request, resp *restful.Response) {
	if err := hdl.resource.Delete(req.Request.Context(), req.PathParameter(hdl.resourceName)); err != nil {
		hdl.Error(req, resp, err)
		return
	}
	if err := resp.WriteEntity(apis.Status{
		ObjectMeta: apis.ObjectMeta{Kind: "Status"},
		Code:       http.StatusOK,
		Status:     apis.StatusSuccess,
		Reason:     "",
		Message:    "",
	}); err != nil {
		hdl.Error(req, resp, err)
		return
	}
}

func (hdl *Handler[T, PT]) Create(req *restful.Request, resp *restful.Response) {
	var t PT = new(T)
	if err := req.ReadEntity(t); err != nil {
		hdl.Error(req, resp, err)
		return
	}
	obj, err := hdl.resource.Create(req.Request.Context(), t)
	if err != nil {
		hdl.Error(req, resp, err)
		return
	}
	if err := resp.WriteEntity(obj); err != nil {
		hdl.Error(req, resp, err)
		return
	}
}

func (hdl *Handler[T, PT]) AddToContainer(container *restful.Container) {
	ws := new(restful.WebService)
	keyParam := restful.PathParameter(hdl.resourceName, "the resource "+hdl.resourceName).DataType("string")

	ws.Path("/" + hdl.resource.Name()).
		ApiVersion(hdl.resource.Version()).
		Doc("API for " + hdl.resource.Version() + "/" + hdl.resource.Name()).
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	// get
	ws.Route(ws.GET(fmt.Sprintf("/{%s}", hdl.resourceName)).
		To(hdl.Get).
		Param(keyParam))

	// update
	ws.Route(ws.PUT(fmt.Sprintf("/{%s}", hdl.resourceName)).
		To(hdl.Update).
		Param(keyParam))

	// delete
	ws.Route(ws.DELETE(fmt.Sprintf("/{%s}", hdl.resourceName)).
		To(hdl.Delete).
		Param(keyParam))

	// list
	ws.Route(ws.GET("/").
		To(hdl.List)).
		Param(restful.QueryParameter("limit", "the limit size").DataType("int")).
		Param(restful.QueryParameter("offset", "the offset").DataType("int")).
		Param(restful.QueryParameter("selector", "selector expression").DataType("string"))

	// create
	ws.Route(ws.POST("/").
		To(hdl.Create))

	container.Add(ws)
}
