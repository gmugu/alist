package quark_union

import (
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/op"
)

type Addition struct {
	OrderBy        string `json:"order_by" type:"select" options:"none,file_type,file_name,updated_at" default:"none"`
	OrderDirection string `json:"order_direction" type:"select" options:"asc,desc" default:"asc"`

	Cookie string `json:"cookie" required:"true"`
	driver.RootID
	// define other
	RefreshToken string `json:"refresh_token" required:"false" default:""`
	// 必要且影响登录,由签名决定
	DeviceID string `json:"device_id"  required:"false" default:""`
	// 登陆所用的数据 无需手动填写
	QueryToken string `json:"query_token" required:"false" default:""`
}

type Conf struct {
	ua      string
	referer string
	api     string
	pr      string

	apiTv    string
	clientID string
	signKey  string
	appVer   string
	channel  string
	codeApi  string
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &QuarkOrUCUnion{
			config: driver.Config{
				Name:              "QuarkUnion",
				OnlyLocal:         false,
				DefaultRoot:       "0",
				NoOverwriteUpload: true,
			},
			conf: Conf{
				ua:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) quark-cloud-drive/2.5.20 Chrome/100.0.4896.160 Electron/18.3.5.4-b478491100 Safari/537.36 Channel/pckk_other_ch",
				referer: "https://pan.quark.cn",
				api:     "https://drive.quark.cn/1/clouddrive",
				pr:      "ucpro",

				apiTv:    "https://open-api-drive.quark.cn",
				clientID: "d3194e61504e493eb6222857bccfed94",
				signKey:  "kw2dvtd7p4t3pjl2d9ed9yc8yej8kw2d",
				appVer:   "1.5.6",
				channel:  "CP",
				codeApi:  "http://api.extscreen.com/quarkdrive",
			},
		}
	})
	op.RegisterDriver(func() driver.Driver {
		return &QuarkOrUCUnion{
			config: driver.Config{
				Name:              "UCUnion",
				OnlyLocal:         false,
				DefaultRoot:       "0",
				NoOverwriteUpload: true,
			},
			conf: Conf{
				ua:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) uc-cloud-drive/2.5.20 Chrome/100.0.4896.160 Electron/18.3.5.4-b478491100 Safari/537.36 Channel/pckk_other_ch",
				referer: "https://drive.uc.cn",
				api:     "https://pc-api.uc.cn/1/clouddrive",
				pr:      "UCBrowser",

				apiTv:    "https://open-api-drive.uc.cn",
				clientID: "5acf882d27b74502b7040b0c65519aa7",
				signKey:  "l3srvtd7p42l0d0x1u8d7yc8ye9kki4d",
				appVer:   "1.6.5",
				channel:  "UCTVOFFICIALWEB",
				codeApi:  "http://api.extscreen.com/ucdrive",
			},
		}
	})
}
