package quark_union

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alist-org/alist/v3/drivers/base"
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

type QuarkOrUCUnion struct {
	*QuarkUCTVCommon
	model.Storage
	Addition
	config driver.Config
	conf   Conf
}

func (d *QuarkOrUCUnion) Config() driver.Config {
	return d.config
}

func (d *QuarkOrUCUnion) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *QuarkOrUCUnion) Init(ctx context.Context) error {
	if d.Addition.DeviceID == "" {
		d.Addition.DeviceID = utils.GetMD5EncodeStr(time.Now().String())
	}
	op.MustSaveDriverStorage(d)

	if d.QuarkUCTVCommon == nil {
		d.QuarkUCTVCommon = &QuarkUCTVCommon{
			AccessToken: "",
		}
	}
	ctx1, cancelFunc := context.WithTimeout(ctx, 5*time.Second)
	defer cancelFunc()
	if d.Addition.RefreshToken == "" {
		if d.Addition.QueryToken == "" {
			qrData, err := d.getLoginCode(ctx1)
			if err != nil {
				return err
			}
			// 展示二维码
			qrTemplate := `<body>
        <img src="data:image/jpeg;base64,%s"/>
    </body>`
			qrPage := fmt.Sprintf(qrTemplate, qrData)
			return fmt.Errorf("need verify: \n%s", qrPage)
		} else {
			// 通过query token获取code -> refresh token
			code, err := d.getCode(ctx1)
			if err != nil {
				return err
			}
			// 通过code获取refresh token
			err = d.getRefreshTokenByTV(ctx1, code, false)
			if err != nil {
				return err
			}
		}
	}
	// 通过refresh token获取access token
	if d.QuarkUCTVCommon.AccessToken == "" {
		err := d.getRefreshTokenByTV(ctx1, d.Addition.RefreshToken, true)
		if err != nil {
			return err
		}
	}

	// 验证 access token 是否有效
	_, err := d.isLogin(ctx1)
	if err != nil {
		return err
	}

	_, err = d.request("/config", http.MethodGet, nil, nil)
	return err
}

func (d *QuarkOrUCUnion) Drop(ctx context.Context) error {
	return nil
}

func (d *QuarkOrUCUnion) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	files, err := d.GetFiles(dir.GetID())
	if err != nil {
		return nil, err
	}
	return utils.SliceConvert(files, func(src File) (model.Obj, error) {
		return fileToObj(src), nil
	})
}

func (d *QuarkOrUCUnion) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	files := &model.Link{}
	var fileLink FileLink
	_, err := d.requestTv(ctx, "/file", "GET", func(req *resty.Request) {
		req.SetQueryParams(map[string]string{
			"method":     "download",
			"group_by":   "source",
			"fid":        file.GetID(),
			"resolution": "low,normal,high,super,2k,4k",
			"support":    "dolby_vision",
		})
	}, &fileLink)
	if err != nil {
		return nil, err
	}
	files.URL = fileLink.Data.DownloadURL
	return files, nil
}

func (d *QuarkOrUCUnion) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) error {
	data := base.Json{
		"dir_init_lock": false,
		"dir_path":      "",
		"file_name":     dirName,
		"pdir_fid":      parentDir.GetID(),
	}
	_, err := d.request("/file", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, nil)
	if err == nil {
		time.Sleep(time.Second)
	}
	return err
}

func (d *QuarkOrUCUnion) Move(ctx context.Context, srcObj, dstDir model.Obj) error {
	data := base.Json{
		"action_type":  1,
		"exclude_fids": []string{},
		"filelist":     []string{srcObj.GetID()},
		"to_pdir_fid":  dstDir.GetID(),
	}
	_, err := d.request("/file/move", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, nil)
	return err
}

func (d *QuarkOrUCUnion) Rename(ctx context.Context, srcObj model.Obj, newName string) error {
	data := base.Json{
		"fid":       srcObj.GetID(),
		"file_name": newName,
	}
	_, err := d.request("/file/rename", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, nil)
	return err
}

func (d *QuarkOrUCUnion) Copy(ctx context.Context, srcObj, dstDir model.Obj) error {
	return errs.NotSupport
}

func (d *QuarkOrUCUnion) Remove(ctx context.Context, obj model.Obj) error {
	data := base.Json{
		"action_type":  1,
		"exclude_fids": []string{},
		"filelist":     []string{obj.GetID()},
	}
	_, err := d.request("/file/delete", http.MethodPost, func(req *resty.Request) {
		req.SetBody(data)
	}, nil)
	return err
}

func (d *QuarkOrUCUnion) Put(ctx context.Context, dstDir model.Obj, stream model.FileStreamer, up driver.UpdateProgress) error {
	tempFile, err := stream.CacheFullInTempFile()
	if err != nil {
		return err
	}
	defer func() {
		_ = tempFile.Close()
	}()
	m := md5.New()
	_, err = io.Copy(m, tempFile)
	if err != nil {
		return err
	}
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	md5Str := hex.EncodeToString(m.Sum(nil))
	s := sha1.New()
	_, err = io.Copy(s, tempFile)
	if err != nil {
		return err
	}
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	sha1Str := hex.EncodeToString(s.Sum(nil))
	// pre
	pre, err := d.upPre(stream, dstDir.GetID())
	if err != nil {
		return err
	}
	log.Debugln("hash: ", md5Str, sha1Str)
	// hash
	finish, err := d.upHash(md5Str, sha1Str, pre.Data.TaskId)
	if err != nil {
		return err
	}
	if finish {
		return nil
	}
	// part up
	partSize := pre.Metadata.PartSize
	var bytes []byte
	md5s := make([]string, 0)
	defaultBytes := make([]byte, partSize)
	total := stream.GetSize()
	left := total
	partNumber := 1
	for left > 0 {
		if utils.IsCanceled(ctx) {
			return ctx.Err()
		}
		if left > int64(partSize) {
			bytes = defaultBytes
		} else {
			bytes = make([]byte, left)
		}
		_, err := io.ReadFull(tempFile, bytes)
		if err != nil {
			return err
		}
		left -= int64(len(bytes))
		log.Debugf("left: %d", left)
		m, err := d.upPart(ctx, pre, stream.GetMimetype(), partNumber, bytes)
		//m, err := driver.UpPart(pre, file.GetMIMEType(), partNumber, bytes, account, md5Str, sha1Str)
		if err != nil {
			return err
		}
		if m == "finish" {
			return nil
		}
		md5s = append(md5s, m)
		partNumber++
		up(int(100 * (total - left) / total))
	}
	err = d.upCommit(pre, md5s)
	if err != nil {
		return err
	}
	return d.upFinish(pre)
}

type QuarkUCTVCommon struct {
	AccessToken string
}

var _ driver.Driver = (*QuarkOrUCUnion)(nil)
