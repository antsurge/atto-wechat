package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

func Code2Session(ctx context.Context, app *miniProgram.Application, code string) error {
	res, err := app.Code2Session(ctx, code)
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.Code2SessionError(res)
	}

	log.Printf("Code2Session response: %+v", res)

	return nil
}
