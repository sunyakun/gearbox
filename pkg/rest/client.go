package rest

import (
	"context"
	"fmt"

	"github.com/sunyakun/gearbox/pkg/apis"
	"github.com/imroc/req/v3"
)

type BodyTransformerFunc func(rawBody []byte, req *req.Request, resp *req.Response) (transformedBody []byte, err error)

type HTTPRestClient[T any, PT interface {
	apis.Object
	*T
}] struct {
	ResourceName string
	C            *req.Client
}

func NewHTTPRestClient[T any, PT interface {
	apis.Object
	*T
}](
	resourceName, baseurl string,
	respBodyTransformer BodyTransformerFunc,
) *HTTPRestClient[T, PT] {
	c := req.C().
		SetResponseBodyTransformer(respBodyTransformer).
		SetBaseURL(baseurl).
		SetCommonHeader("Content-Type", "application/json")

	return &HTTPRestClient[T, PT]{
		ResourceName: resourceName,
		C:            c,
	}
}

func (cli *HTTPRestClient[T, PT]) Get(ctx context.Context, key string) (PT, error) {
	var t T
	_, err := cli.C.R().SetSuccessResult(&t).Get(fmt.Sprintf("%s/%s", cli.ResourceName, key))
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (cli *HTTPRestClient[T, PT]) GetList(ctx context.Context, opts apis.ListOptions) ([]PT, int64, error) {
	var objList apis.ObjectList[PT]
	_, err := cli.C.R().SetSuccessResult(&objList).Get(cli.ResourceName)
	if err != nil {
		return nil, 0, err
	}
	return objList.Items, objList.Count, nil
}

func (cli *HTTPRestClient[T, PT]) Create(ctx context.Context, obj PT) (PT, error) {
	var t T
	_, err := cli.C.R().SetSuccessResult(&t).SetBody(obj).Post(cli.ResourceName)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (cli *HTTPRestClient[T, PT]) Update(ctx context.Context, key string, obj PT) error {
	var t T
	_, err := cli.C.R().SetSuccessResult(&t).SetBody(obj).Put(fmt.Sprintf("%s/%s", cli.ResourceName, key))
	if err != nil {
		return err
	}
	return nil
}

func (cli *HTTPRestClient[T, PT]) Delete(ctx context.Context, key string) error {
	_, err := cli.C.R().Delete(fmt.Sprintf("%s/%s", cli.ResourceName, key))
	if err != nil {
		return err
	}
	return nil
}
