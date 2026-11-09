package miniProgram

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/antsurge/atto-wechat/core/errors"
)

// newLiveApp starts a fake server whose replies for the live broadcast
// endpoints are the ones given, keyed by path.
func newLiveApp(t *testing.T, replies map[string]string) (*Application, *fakeWechat) {
	t.Helper()

	fake := &fakeWechat{
		tokenBody:      `{"access_token":"at","expires_in":7200}`,
		messageReplies: replies,
	}

	srv := newFakeServer(fake)
	t.Cleanup(srv.Close)

	return newApp(t, srv.URL, nil), fake
}

// liveRoomParams is a valid CreateLiveRoomParams.
func liveRoomParams() CreateLiveRoomParams {
	return CreateLiveRoomParams{
		Name:         "直播间",
		CoverImg:     "media-cover",
		StartTime:    1735689600,
		EndTime:      1735693200,
		AnchorName:   "主播",
		AnchorWechat: "anchor",
		ShareImg:     "media-share",
		FeedsImg:     "media-feeds",
		LiveType:     LiveRoomTypePush,
	}
}

// liveGoodsParams is a valid AddLiveGoodsParams.
func liveGoodsParams() AddLiveGoodsParams {
	return AddLiveGoodsParams{
		GoodsInfo: LiveGoodsInfo{
			CoverImgURL: "media-goods",
			Name:        "商品",
			PriceType:   LiveGoodsPriceTypeFixed,
			Price:       9.9,
			URL:         "pages/goods/detail?id=1",
		},
	}
}

// TestLiveBroadcastEndpoints checks the method, path and token of every live
// broadcast endpoint, so a mistyped route or a GET sent as POST fails here
// instead of against WeChat.
func TestLiveBroadcastEndpoints(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		method string
		call   func(context.Context, *Application) error
	}{
		{
			name:   "CreateLiveRoom",
			path:   createLiveRoomPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.CreateLiveRoom(ctx, liveRoomParams())
				return err
			},
		},
		{
			name:   "EditLiveRoom",
			path:   editLiveRoomPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				params := EditLiveRoomParams{
					RoomID:       1,
					Name:         "直播间",
					CoverImg:     "media-cover",
					StartTime:    1735689600,
					EndTime:      1735693200,
					AnchorName:   "主播",
					AnchorWechat: "anchor",
					ShareImg:     "media-share",
					FeedsImg:     "media-feeds",
				}
				_, err := app.EditLiveRoom(ctx, params)
				return err
			},
		},
		{
			name:   "GetLiveRoomList",
			path:   getLiveRoomListPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveRoomList(ctx, GetLiveRoomListParams{Limit: 10})
				return err
			},
		},
		{
			name:   "DeleteLiveRoom",
			path:   deleteLiveRoomPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.DeleteLiveRoom(ctx, DeleteLiveRoomParams{RoomID: 1})
				return err
			},
		},
		{
			name:   "GetLiveRoomPushURL",
			path:   getLiveRoomPushURLPath,
			method: "GET",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveRoomPushURL(ctx, GetLiveRoomPushURLParams{RoomID: 1})
				return err
			},
		},
		{
			name:   "GetLiveRoomSharedCode",
			path:   getLiveRoomSharedCodePath,
			method: "GET",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveRoomSharedCode(ctx, GetLiveRoomSharedCodeParams{RoomID: 1})
				return err
			},
		},
		{
			name:   "GetSubAnchor",
			path:   getSubAnchorPath,
			method: "GET",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetSubAnchor(ctx, GetSubAnchorParams{RoomID: 1})
				return err
			},
		},
		{
			name:   "AddSubAnchor",
			path:   addSubAnchorPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.AddSubAnchor(ctx, AddSubAnchorParams{RoomID: 1, Username: "sub"})
				return err
			},
		},
		{
			name:   "ModifySubAnchor",
			path:   modifySubAnchorPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.ModifySubAnchor(ctx, ModifySubAnchorParams{RoomID: 1, Username: "sub"})
				return err
			},
		},
		{
			name:   "DeleteSubAnchor",
			path:   deleteSubAnchorPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.DeleteSubAnchor(ctx, DeleteSubAnchorParams{RoomID: 1})
				return err
			},
		},
		{
			name:   "ImportLiveRoomGoods",
			path:   importLiveRoomGoodsPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.ImportLiveRoomGoods(ctx, ImportLiveRoomGoodsParams{
					RoomID:   1,
					GoodsIDs: []int64{2},
				})
				return err
			},
		},
		{
			name:   "DeleteLiveRoomGoods",
			path:   deleteLiveRoomGoodsPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.DeleteLiveRoomGoods(ctx, DeleteLiveRoomGoodsParams{
					RoomID:  1,
					GoodsID: 2,
				})
				return err
			},
		},
		{
			name:   "PushLiveRoomGoods",
			path:   pushLiveRoomGoodsPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.PushLiveRoomGoods(ctx, PushLiveRoomGoodsParams{
					RoomID:  1,
					GoodsID: 2,
				})
				return err
			},
		},
		{
			name:   "SaleLiveRoomGoods",
			path:   saleLiveRoomGoodsPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.SaleLiveRoomGoods(ctx, SaleLiveRoomGoodsParams{
					RoomID:  1,
					GoodsID: 2,
					OnSale:  1,
				})
				return err
			},
		},
		{
			name:   "SortLiveRoomGoods",
			path:   sortLiveRoomGoodsPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.SortLiveRoomGoods(ctx, SortLiveRoomGoodsParams{
					RoomID: 1,
					Goods:  []LiveGoodsOrder{{GoodsID: 2}, {GoodsID: 3}},
				})
				return err
			},
		},
		{
			name:   "AddLiveRoomAssistant",
			path:   addLiveRoomAssistantPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.AddLiveRoomAssistant(ctx, AddLiveRoomAssistantParams{
					RoomID: 1,
					Users:  []LiveRoomAssistant{{Username: "helper", Nickname: "小助手"}},
				})
				return err
			},
		},
		{
			name:   "ModifyLiveRoomAssistant",
			path:   modifyLiveRoomAssistantPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.ModifyLiveRoomAssistant(ctx, ModifyLiveRoomAssistantParams{
					RoomID:   1,
					Username: "helper",
					Nickname: "小助手",
				})
				return err
			},
		},
		{
			name:   "GetLiveRoomAssistantList",
			path:   getLiveRoomAssistantListPath,
			method: "GET",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveRoomAssistantList(ctx, GetLiveRoomAssistantListParams{RoomID: 1})
				return err
			},
		},
		{
			name:   "RemoveLiveRoomAssistant",
			path:   removeLiveRoomAssistantPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.RemoveLiveRoomAssistant(ctx, RemoveLiveRoomAssistantParams{
					RoomID:   1,
					Username: "helper",
				})
				return err
			},
		},
		{
			name:   "UpdateLiveRoomComment",
			path:   updateLiveRoomCommentPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.UpdateLiveRoomComment(ctx, UpdateLiveRoomCommentParams{
					RoomID:     1,
					BanComment: 1,
				})
				return err
			},
		},
		{
			name:   "UpdateLiveRoomFeedPublic",
			path:   updateLiveRoomFeedPublicPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.UpdateLiveRoomFeedPublic(ctx, UpdateLiveRoomFeedPublicParams{
					RoomID:        1,
					IsFeedsPublic: 1,
				})
				return err
			},
		},
		{
			name:   "UpdateLiveRoomKF",
			path:   updateLiveRoomKFPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.UpdateLiveRoomKF(ctx, UpdateLiveRoomKFParams{
					RoomID:  1,
					CloseKf: 1,
				})
				return err
			},
		},
		{
			name:   "UpdateLiveRoomReplay",
			path:   updateLiveRoomReplayPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.UpdateLiveRoomReplay(ctx, UpdateLiveRoomReplayParams{
					RoomID:      1,
					CloseReplay: 1,
				})
				return err
			},
		},
		{
			name:   "SetLiveGoodsKey",
			path:   setLiveGoodsKeyPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.SetLiveGoodsKey(ctx, SetLiveGoodsKeyParams{
					GoodsKey: []string{"key"},
				})
				return err
			},
		},
		{
			name:   "GetLiveGoodsKey",
			path:   getLiveGoodsKeyPath,
			method: "GET",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveGoodsKey(ctx)
				return err
			},
		},
		{
			name:   "DownloadLiveGoodsVideo",
			path:   downloadLiveGoodsVideoPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.DownloadLiveGoodsVideo(ctx, DownloadLiveGoodsVideoParams{
					RoomID:  1,
					GoodsID: 2,
				})
				return err
			},
		},
		{
			name:   "AddLiveGoods",
			path:   addLiveGoodsPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.AddLiveGoods(ctx, liveGoodsParams())
				return err
			},
		},
		{
			name:   "UpdateLiveGoods",
			path:   updateLiveGoodsPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				params := liveGoodsParams()
				params.GoodsInfo.GoodsID = 2
				_, err := app.UpdateLiveGoods(ctx, UpdateLiveGoodsParams(params))
				return err
			},
		},
		{
			name:   "ResubmitLiveGoodsAudit",
			path:   resubmitLiveGoodsAuditPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.ResubmitLiveGoodsAudit(ctx, ResubmitLiveGoodsAuditParams{GoodsID: 2})
				return err
			},
		},
		{
			name:   "ResetLiveGoodsAudit",
			path:   resetLiveGoodsAuditPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.ResetLiveGoodsAudit(ctx, ResetLiveGoodsAuditParams{
					GoodsID: 2,
					AuditID: 3,
				})
				return err
			},
		},
		{
			name:   "GetLiveGoodsAuditInfo",
			path:   getLiveGoodsAuditInfoPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveGoodsAuditInfo(ctx, GetLiveGoodsAuditInfoParams{
					GoodsIDs: []int64{2},
				})
				return err
			},
		},
		{
			name:   "GetLiveGoodsList",
			path:   getLiveGoodsListPath,
			method: "GET",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveGoodsList(ctx, GetLiveGoodsListParams{
					Limit:  10,
					Status: LiveGoodsAuditApproved,
				})
				return err
			},
		},
		{
			name:   "DeleteLiveGoods",
			path:   deleteLiveGoodsPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.DeleteLiveGoods(ctx, DeleteLiveGoodsParams{GoodsID: 2})
				return err
			},
		},
		{
			name:   "AddLiveRole",
			path:   addLiveRolePath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.AddLiveRole(ctx, AddLiveRoleParams{
					Username: "anchor",
					Role:     LiveRoleAnchor,
				})
				return err
			},
		},
		{
			name:   "DeleteLiveRole",
			path:   deleteLiveRolePath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.DeleteLiveRole(ctx, DeleteLiveRoleParams{
					Username: "anchor",
					Role:     LiveRoleAnchor,
				})
				return err
			},
		},
		{
			name:   "GetLiveRoleList",
			path:   getLiveRoleListPath,
			method: "GET",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveRoleList(ctx, GetLiveRoleListParams{Limit: 10})
				return err
			},
		},
		{
			name:   "PushLiveMessage",
			path:   pushLiveMessagePath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.PushLiveMessage(ctx, PushLiveMessageParams{
					RoomID:      1,
					UserOpenIDs: []string{"openid-1"},
				})
				return err
			},
		},
		{
			name:   "GetLiveFollowers",
			path:   getLiveFollowersPath,
			method: "POST",
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveFollowers(ctx, GetLiveFollowersParams{Limit: 100})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, fake := newLiveApp(t, nil)

			if err := tt.call(context.Background(), app); err != nil {
				t.Fatalf("err = %v", err)
			}

			if got := fake.lastMessageMethod[tt.path]; got != tt.method {
				t.Fatalf("method = %q, want %q", got, tt.method)
			}
			if got := fake.lastMessageQuery[tt.path].Get("access_token"); got != "at" {
				t.Fatalf("access_token = %q, want %q", got, "at")
			}
		})
	}
}

// TestLiveBroadcastBodies pins the JSON of the endpoints whose body is the
// contract: the field names WeChat expects are not the Go ones, so they are
// worth asserting literally.
func TestLiveBroadcastBodies(t *testing.T) {
	t.Run("CreateLiveRoom", func(t *testing.T) {
		app, fake := newLiveApp(t, map[string]string{
			createLiveRoomPath: `{"errcode":0,"errmsg":"ok","roomId":9,"qrcode_url":"https://x"}`,
		})

		params := liveRoomParams()
		params.IsFeedsPublic = IntPtr(0)
		params.CloseKf = IntPtr(1)

		result, err := app.CreateLiveRoom(context.Background(), params)
		if err != nil {
			t.Fatal(err)
		}
		if !result.OK() || result.RoomID != 9 || result.QRCodeURL != "https://x" {
			t.Fatalf("response = %+v", result)
		}

		assertLiveBody(t, fake.lastMessageBody[createLiveRoomPath], map[string]any{
			"name":          "直播间",
			"coverImg":      "media-cover",
			"startTime":     float64(1735689600),
			"endTime":       float64(1735693200),
			"anchorName":    "主播",
			"anchorWechat":  "anchor",
			"shareImg":      "media-share",
			"feedsImg":      "media-feeds",
			"isFeedsPublic": float64(0),
			"type":          float64(LiveRoomTypePush),
			"closeLike":     float64(0),
			"closeGoods":    float64(0),
			"closeComment":  float64(0),
			"closeKf":       float64(1),
		})
	})

	t.Run("GetLiveRoomListReplay", func(t *testing.T) {
		app, fake := newLiveApp(t, map[string]string{
			getLiveRoomListPath: `{"errcode":0,"errmsg":"ok","live_replay":[{"create_time":"1","expire_time":"2","media_url":"https://v"}],"total":1}`,
		})

		result, err := app.GetLiveRoomList(context.Background(), GetLiveRoomListParams{
			Limit:  10,
			Action: LiveRoomActionGetReplay,
			RoomID: 7,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.LiveReplay) != 1 || result.LiveReplay[0].MediaURL != "https://v" {
			t.Fatalf("response = %+v", result)
		}

		assertLiveBody(t, fake.lastMessageBody[getLiveRoomListPath], map[string]any{
			"start":   float64(0),
			"limit":   float64(10),
			"action":  LiveRoomActionGetReplay,
			"room_id": float64(7),
		})
	})

	t.Run("UpdateLiveRoomComment", func(t *testing.T) {
		app, fake := newLiveApp(t, nil)

		_, err := app.UpdateLiveRoomComment(context.Background(), UpdateLiveRoomCommentParams{
			RoomID:     8,
			BanComment: 1,
		})
		if err != nil {
			t.Fatal(err)
		}

		// WeChat names the room id field id on this endpoint, not roomId.
		assertLiveBody(t, fake.lastMessageBody[updateLiveRoomCommentPath], map[string]any{
			"id":         float64(8),
			"banComment": float64(1),
		})
	})

	t.Run("ImportLiveRoomGoods", func(t *testing.T) {
		app, fake := newLiveApp(t, nil)

		_, err := app.ImportLiveRoomGoods(context.Background(), ImportLiveRoomGoodsParams{
			RoomID:   1,
			GoodsIDs: []int64{2, 3},
		})
		if err != nil {
			t.Fatal(err)
		}

		assertLiveBody(t, fake.lastMessageBody[importLiveRoomGoodsPath], map[string]any{
			"roomId": float64(1),
			"ids":    []any{float64(2), float64(3)},
		})
	})

	t.Run("AddLiveGoods", func(t *testing.T) {
		app, fake := newLiveApp(t, map[string]string{
			addLiveGoodsPath: `{"errcode":0,"errmsg":"ok","goodsId":5,"auditId":6}`,
		})

		result, err := app.AddLiveGoods(context.Background(), liveGoodsParams())
		if err != nil {
			t.Fatal(err)
		}
		if !result.OK() || result.GoodsID != 5 || result.AuditID != 6 {
			t.Fatalf("response = %+v", result)
		}

		// price2 is omitted for a fixed price: WeChat reads its presence as a
		// range or a discount.
		assertLiveBody(t, fake.lastMessageBody[addLiveGoodsPath], map[string]any{
			"goodsInfo": map[string]any{
				"coverImgUrl": "media-goods",
				"name":        "商品",
				"priceType":   float64(LiveGoodsPriceTypeFixed),
				"price":       9.9,
				"url":         "pages/goods/detail?id=1",
			},
		})
	})

	t.Run("PushLiveMessage", func(t *testing.T) {
		app, fake := newLiveApp(t, map[string]string{
			pushLiveMessagePath: `{"errcode":0,"errmsg":"ok","message_id":"m-1"}`,
		})

		result, err := app.PushLiveMessage(context.Background(), PushLiveMessageParams{
			RoomID:      1,
			UserOpenIDs: []string{"openid-1", "openid-2"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.MessageID != "m-1" {
			t.Fatalf("response = %+v", result)
		}

		assertLiveBody(t, fake.lastMessageBody[pushLiveMessagePath], map[string]any{
			"room_id":     float64(1),
			"user_openid": []any{"openid-1", "openid-2"},
		})
	})

	t.Run("GetLiveFollowers", func(t *testing.T) {
		app, fake := newLiveApp(t, map[string]string{
			getLiveFollowersPath: `{"errcode":0,"errmsg":"ok","followers":[{"openid":"o","create_time":1}],"page_break":"p2"}`,
		})

		result, err := app.GetLiveFollowers(context.Background(), GetLiveFollowersParams{
			Limit:     100,
			PageBreak: "p1",
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.PageBreak != "p2" || len(result.Followers) != 1 {
			t.Fatalf("response = %+v", result)
		}

		assertLiveBody(t, fake.lastMessageBody[getLiveFollowersPath], map[string]any{
			"limit":      float64(100),
			"page_break": "p1",
		})
	})
}

// TestLiveBroadcastQueries pins the query of the GET endpoints.
func TestLiveBroadcastQueries(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		call   func(context.Context, *Application) error
		expect map[string]string
		absent []string
	}{
		{
			name: "GetLiveRoomPushURL",
			path: getLiveRoomPushURLPath,
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveRoomPushURL(ctx, GetLiveRoomPushURLParams{RoomID: 12})
				return err
			},
			expect: map[string]string{"roomId": "12"},
		},
		{
			name: "GetLiveRoomSharedCode",
			path: getLiveRoomSharedCodePath,
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveRoomSharedCode(ctx, GetLiveRoomSharedCodeParams{
					RoomID: 12,
				})
				return err
			},
			expect: map[string]string{"roomId": "12"},
			absent: []string{"params"},
		},
		{
			name: "GetLiveGoodsList",
			path: getLiveGoodsListPath,
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveGoodsList(ctx, GetLiveGoodsListParams{
					Limit:  20,
					Status: LiveGoodsAuditRejected,
				})
				return err
			},
			expect: map[string]string{
				"offset": "0",
				"limit":  "20",
				"status": "3",
			},
		},
		{
			name: "GetLiveRoleListWithoutKeyword",
			path: getLiveRoleListPath,
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveRoleList(ctx, GetLiveRoleListParams{Limit: 20})
				return err
			},
			expect: map[string]string{"offset": "0", "limit": "20"},
			absent: []string{"keyword"},
		},
		{
			name: "GetLiveRoleListWithKeyword",
			path: getLiveRoleListPath,
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveRoleList(ctx, GetLiveRoleListParams{
					Limit:   20,
					Keyword: "anchor",
				})
				return err
			},
			expect: map[string]string{"keyword": "anchor"},
		},
		{
			name: "GetLiveGoodsKeyWithoutQuery",
			path: getLiveGoodsKeyPath,
			call: func(ctx context.Context, app *Application) error {
				_, err := app.GetLiveGoodsKey(ctx)
				return err
			},
			expect: map[string]string{},
			absent: []string{"roomId", "goodsId"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, fake := newLiveApp(t, nil)

			if err := tt.call(context.Background(), app); err != nil {
				t.Fatalf("err = %v", err)
			}

			query := fake.lastMessageQuery[tt.path]
			for key, want := range tt.expect {
				if got := query.Get(key); got != want {
					t.Fatalf("%s = %q, want %q", key, got, want)
				}
			}
			for _, key := range tt.absent {
				if _, ok := query[key]; ok {
					t.Fatalf("%s = %v, want it absent", key, query[key])
				}
			}
		})
	}
}

// TestLiveBroadcastErrors checks that the errcode from the reply is turned into
// the documented description, without leaking WeChat's raw errmsg.
func TestLiveBroadcastErrors(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		reply  string
		call   func(context.Context, *Application) error
		expect string
	}{
		{
			name:  "CreateLiveRoomAnchorNotVerified",
			path:  createLiveRoomPath,
			reply: `{"errcode":300036,"errmsg":"raw text rid: 1"}`,
			call: func(ctx context.Context, app *Application) error {
				result, err := app.CreateLiveRoom(ctx, liveRoomParams())
				if err != nil {
					return err
				}
				return CreateLiveRoomError(result)
			},
			expect: "主播微信号未实名认证",
		},
		{
			name:  "AddSubAnchorDuplicate",
			path:  addSubAnchorPath,
			reply: `{"errcode":500004,"errmsg":"raw text rid: 1"}`,
			call: func(ctx context.Context, app *Application) error {
				result, err := app.AddSubAnchor(ctx, AddSubAnchorParams{RoomID: 1, Username: "sub"})
				if err != nil {
					return err
				}
				return AddSubAnchorError(result)
			},
			expect: "不能设置重复的副号",
		},
		{
			name:  "PushLiveRoomGoodsInFlight",
			path:  pushLiveRoomGoodsPath,
			reply: `{"errcode":300047,"errmsg":"raw text rid: 1"}`,
			call: func(ctx context.Context, app *Application) error {
				result, err := app.PushLiveRoomGoods(ctx, PushLiveRoomGoodsParams{RoomID: 1, GoodsID: 2})
				if err != nil {
					return err
				}
				return PushLiveRoomGoodsError(result)
			},
			expect: "已有商品正在推送，请稍后再试",
		},
		{
			name:  "PushLiveMessageNotLive",
			path:  pushLiveMessagePath,
			reply: `{"errcode":9410006,"errmsg":"raw text rid: 1"}`,
			call: func(ctx context.Context, app *Application) error {
				result, err := app.PushLiveMessage(ctx, PushLiveMessageParams{
					RoomID:      1,
					UserOpenIDs: []string{"o"},
				})
				if err != nil {
					return err
				}
				return PushLiveMessageError(result)
			},
			expect: "推送的直播间非直播中状态",
		},
		{
			name:  "GetLiveRoomListSystemBusy",
			path:  getLiveRoomListPath,
			reply: `{"errcode":-1,"errmsg":"raw text rid: 1"}`,
			call: func(ctx context.Context, app *Application) error {
				result, err := app.GetLiveRoomList(ctx, GetLiveRoomListParams{Limit: 1})
				if err != nil {
					return err
				}
				return GetLiveRoomListError(result)
			},
			expect: "系统繁忙，请稍后再试",
		},
		{
			name:  "GetLiveGoodsListUndocumentedCode",
			path:  getLiveGoodsListPath,
			reply: `{"errcode":999999,"errmsg":"raw text rid: 1"}`,
			call: func(ctx context.Context, app *Application) error {
				result, err := app.GetLiveGoodsList(ctx, GetLiveGoodsListParams{
					Limit:  1,
					Status: LiveGoodsAuditApproved,
				})
				if err != nil {
					return err
				}
				return GetLiveGoodsListError(result)
			},
			expect: "系统错误，请稍后再试",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := newLiveApp(t, map[string]string{tt.path: tt.reply})

			err := tt.call(context.Background(), app)
			if err == nil {
				t.Fatal("err = nil, want the documented error")
			}

			if !strings.Contains(err.Error(), tt.expect) {
				t.Fatalf("err = %q, want %q", err, tt.expect)
			}
			if strings.Contains(err.Error(), "raw text") || strings.Contains(err.Error(), "rid:") {
				t.Fatalf("err = %q, want the raw errmsg to be dropped", err)
			}
		})
	}
}

// TestLiveBroadcastValidation checks the params that are refused before
// anything is sent, so a caller gets a local error instead of a round trip.
func TestLiveBroadcastValidation(t *testing.T) {
	app, fake := newLiveApp(t, nil)

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "CreateLiveRoomWithoutName",
			call: func() error {
				params := liveRoomParams()
				params.Name = ""
				_, err := app.CreateLiveRoom(context.Background(), params)
				return err
			},
		},
		{
			name: "CreateLiveRoomEndBeforeStart",
			call: func() error {
				params := liveRoomParams()
				params.EndTime = params.StartTime
				_, err := app.CreateLiveRoom(context.Background(), params)
				return err
			},
		},
		{
			name: "CreateLiveRoomUnknownType",
			call: func() error {
				params := liveRoomParams()
				params.LiveType = 9
				_, err := app.CreateLiveRoom(context.Background(), params)
				return err
			},
		},
		{
			name: "GetLiveRoomListWithoutLimit",
			call: func() error {
				_, err := app.GetLiveRoomList(context.Background(), GetLiveRoomListParams{})
				return err
			},
		},
		{
			name: "GetLiveRoomListReplayWithoutRoom",
			call: func() error {
				_, err := app.GetLiveRoomList(context.Background(), GetLiveRoomListParams{
					Limit:  1,
					Action: LiveRoomActionGetReplay,
				})
				return err
			},
		},
		{
			name: "SaleLiveRoomGoodsUnknownState",
			call: func() error {
				_, err := app.SaleLiveRoomGoods(context.Background(), SaleLiveRoomGoodsParams{
					RoomID:  1,
					GoodsID: 2,
					OnSale:  2,
				})
				return err
			},
		},
		{
			name: "UpdateLiveRoomKFUnknownState",
			call: func() error {
				_, err := app.UpdateLiveRoomKF(context.Background(), UpdateLiveRoomKFParams{
					RoomID:  1,
					CloseKf: 2,
				})
				return err
			},
		},
		{
			name: "AddLiveGoodsRangeWithoutPrice2",
			call: func() error {
				params := liveGoodsParams()
				params.GoodsInfo.PriceType = LiveGoodsPriceTypeRange
				_, err := app.AddLiveGoods(context.Background(), params)
				return err
			},
		},
		{
			name: "UpdateLiveGoodsWithoutGoodsID",
			call: func() error {
				_, err := app.UpdateLiveGoods(context.Background(), UpdateLiveGoodsParams{
					GoodsInfo: liveGoodsParams().GoodsInfo,
				})
				return err
			},
		},
		{
			name: "GetLiveGoodsListUnknownStatus",
			call: func() error {
				_, err := app.GetLiveGoodsList(context.Background(), GetLiveGoodsListParams{
					Limit:  1,
					Status: 9,
				})
				return err
			},
		},
		{
			name: "AddLiveRoleSuperAdmin",
			call: func() error {
				_, err := app.AddLiveRole(context.Background(), AddLiveRoleParams{
					Username: "anchor",
					Role:     LiveRoleSuperAdmin,
				})
				return err
			},
		},
		{
			name: "DeleteLiveRoleUnknownRole",
			call: func() error {
				_, err := app.DeleteLiveRole(context.Background(), DeleteLiveRoleParams{
					Username: "anchor",
					Role:     9,
				})
				return err
			},
		},
		{
			name: "PushLiveMessageWithoutOpenID",
			call: func() error {
				_, err := app.PushLiveMessage(context.Background(), PushLiveMessageParams{
					RoomID: 1,
				})
				return err
			},
		},
		{
			name: "GetLiveFollowersWithoutLimit",
			call: func() error {
				_, err := app.GetLiveFollowers(context.Background(), GetLiveFollowersParams{})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v, want CodeInvalidArgument", err)
			}
		})
	}

	// A rejected call must not reach WeChat at all.
	if len(fake.lastMessageMethod) != 0 {
		t.Fatalf("requests = %v, want none", fake.lastMessageMethod)
	}
}

// TestGetSubAnchorResponse checks the reply that carries data.
func TestGetSubAnchorResponse(t *testing.T) {
	app, _ := newLiveApp(t, map[string]string{
		getSubAnchorPath: `{"errcode":0,"errmsg":"ok","username":"wx-sub"}`,
	})

	result, err := app.GetSubAnchor(context.Background(), GetSubAnchorParams{RoomID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() || result.Username != "wx-sub" {
		t.Fatalf("response = %+v", result)
	}
}

// assertLiveBody compares a recorded request body with want.
func assertLiveBody(t *testing.T, got []byte, want map[string]any) {
	t.Helper()

	if len(got) == 0 {
		t.Fatal("body is empty, want the request to carry one")
	}

	var body map[string]any
	if err := json.Unmarshal(got, &body); err != nil {
		t.Fatalf("body = %s: %v", got, err)
	}

	if !reflect.DeepEqual(body, want) {
		t.Fatalf("body = %v, want %v", body, want)
	}
}
