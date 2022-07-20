package srv

const (
	ALLOW_OWN = "own" //应用 owner
	ALLOW_DEV = "dev" //开发者，允许上传应用
)

const (
	ABILITY_CLOUD = "cloud" //云函数
)

const (
	APP_VER_STATE_NONE = 0
	APP_VER_STATE_OK   = 1
)

const (
	ERRNO_OK             = 200
	ERRNO_404            = 404
	ERRNO_INPUT_DATA     = 400
	ERRNO_LOGIN_CODE     = 601
	ERRNO_NOT_LOGIN      = 602
	ERRNO_NO_PERMISSIONS = 603
	ERRNO_APP_VER        = 604
)

const (
	SERVICE_REDIS   = "redis"
	SERVICE_MONGODB = "mongodb"
	SERVICE_SMTP    = "smtp"
	SERVICE_USER    = "uv-user"
	SERVICE_APP     = "uv-app"
)

//用户信息
type User struct {
	Id    string `json:"id" title:"用户ID"`
	Email string `json:"email,omitempty" title:"邮箱"`
}

//应用信息
type App struct {
	Id    string `json:"id" title:"应用ID"`
	Title string `json:"title" title:"应用说明"`
}

//应用版本
type AppVer struct {
	Appid string `json:"appid" title:"应用ID"`
	Ver   string `json:"ver" title:"版本号"`
	Title string `json:"title" title:"版本说明"`
}

//应用成员
type AppMember struct {
	Appid string `json:"appid" title:"应用ID"`
	Uid   string `json:"uid" title:"用户ID"`
	Allow string `json:"allow" title:"允许的权限"`
	Title string `json:"title" title:"成员备注名"`
	Email string `json:"email" title:"邮箱"`
}

type Page struct {
	Count      int32 `json:"count,omitempty"`
	P          int32 `json:"p,omitempty"`
	N          int32 `json:"n,omitempty"`
	TotalCount int32 `json:"totalCount,omitempty"`
}

type AppQueryResult struct {
	Page  *Page  `json:"page,omitempty" title:"分页信息"`
	Items []*App `json:"items"`
}

type AppVerQueryResult struct {
	Page  *Page     `json:"page,omitempty" title:"分页信息"`
	Items []*AppVer `json:"items"`
}

type AppMemberQueryResult struct {
	Page  *Page        `json:"page,omitempty" title:"分页信息"`
	Items []*AppMember `json:"items"`
}

// 获取登录验证码
type LoginCodeTask struct {
	Email string `json:"email" title:"邮箱"`
}
type LoginCodeResult struct {
}

//登录
type LoginTask struct {
	Email string `json:"email" title:"邮箱"`
	Code  string `json:"code" title:"验证码"`
}
type LoginResult struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

//退出
type LogoutTask struct {
	Token string `json:"token"`
}
type LogoutResult struct {
}

//获取用户信息
type UserGetTask struct {
	Token string `json:"token"`
}

//创建应用
type MyAppCreateTask struct {
	Token string `json:"token"`
	Title string `json:"title" title:"应用说明"`
}

//查询我的应用
type MyAppQueryTask struct {
	Token string `json:"token"`
	P     int32  `json:"p" title:"分页位置"`
	N     int32  `json:"n" title:"每页限制条数"`
}

//查询应用版本
type AppVerQueryTask struct {
	Token string `json:"token"`
	Appid string `json:"appid" title:"应用ID"`
	P     int32  `json:"p" title:"分页位置"`
	N     int32  `json:"n" title:"每页限制条数"`
}

//查询应用成员
type AppMemberQueryTask struct {
	Token string `json:"token"`
	Appid string `json:"appid" title:"应用ID"`
	P     int32  `json:"p" title:"分页位置"`
	N     int32  `json:"n" title:"每页限制条数"`
}

//添加应用成员
type AppMemberAddTask struct {
	Token string `json:"token"`
	Appid string `json:"appid" title:"应用ID"`
	Email string `json:"email" title:"邮箱"`
	Allow string `json:"allow" title:"允许的权限"`
	Title string `json:"title" title:"成员备注名"`
}

//删除应用成员
type AppMemberRemoveTask struct {
	Token string `json:"token"`
	Appid string `json:"appid" title:"应用ID"`
	Email string `json:"email" title:"邮箱"`
}

//获取应用上传信息
type AppUpTask struct {
	Token   string `json:"token"`
	Appid   string `json:"appid" title:"应用ID"`
	Ver     string `json:"ver" title:"版本号"`
	Ability string `json:"ability" title:"应用包能力"`
}

type AppUpResult struct {
	Url    string            `json:"url" title:"上传URL"`
	Method string            `json:"method" title:"上传方法"`
	Data   map[string]string `json:"data" title:"上传数据"`
	Key    string            `json:"key" title:"上传文件KEY"`
}

//完成应用上传
type AppUpDoneTask struct {
	Token string      `json:"token"`
	Appid string      `json:"appid" title:"应用ID"`
	Ver   string      `json:"ver" title:"版本号"`
	Title string      `json:"title" title:"版本说明"`
	Info  interface{} `json:"info" title:"版本信息"`
}

type AppUpDoneResult struct {
}
