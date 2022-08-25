package srv

const (
	ROLE_OWNER      = "owner"
	ROLE_READ_WRITE = "readwrite"
	ROLE_READ_ONLY  = "readonly"
)

type User struct {
	Id    string `json:"id"`
	Email string `json:"email"`
}

type SendMailTask struct {
	Email string `json:"email"`
}

type LoginTask struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type LoginResult struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

type UserGetTask struct {
	Token string `json:"token"`
}

type Container struct {
	Id     string      `json:"id"`
	Info   interface{} `json:"info,omitempty"`
	Ver    int         `json:"ver"`
	Secret string      `json:"secret"`
}

type ContainerCreateTask struct {
	Token string      `json:"token"`
	Info  interface{} `json:"info"`
}

type ContainerSetTask struct {
	Token  string      `json:"token"`
	Id     string      `json:"id"`
	Info   interface{} `json:"info"`
	Secret bool        `json:"secret"`
}

type ContainerGetTask struct {
	Token string `json:"token"`
	Id    string `json:"id"`
}

type ContainerInfoGetTask struct {
	Sign      string `json:"sign"`
	Id        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
	Ver       int    `json:"ver"`
}

type ContainerInfoGetResult struct {
	Info interface{} `json:"info,omitempty"`
	Ver  int         `json:"ver"`
}

type Member struct {
	Id   string `json:"id"`
	Role string `json:"role"`
}

type ContainerMemberAddTask struct {
	Token string `json:"token"`
	Id    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type ContainerMemberRemoveTask struct {
	Token string `json:"token"`
	Id    string `json:"id"`
	Email string `json:"email"`
}

type ContainerAppGetTask struct {
	Id        string `json:"id"`
	Appid     string `json:"appid"`
	Ver       string `json:"ver"`
	Ability   string `json:"ability"`
	Sign      string `json:"sign"`
	Timestamp int64  `json:"timestamp"`
}

type ContainerAppGetResult struct {
	Info interface{} `json:"info,omitempty"`
	Url  string      `json:"url,omitempty"`
}

type App struct {
	Id   string      `json:"id"`
	Info interface{} `json:"info,omitempty"`
}

type AppCreateTask struct {
	Token string      `json:"token"`
	Info  interface{} `json:"info,omitempty"`
}

type AppGetTask struct {
	Token string `json:"token"`
	Id    string `json:"id"`
}

type AppSetTask struct {
	Token string      `json:"token"`
	Id    string      `json:"id"`
	Info  interface{} `json:"info,omitempty"`
}

type AppVerUpTask struct {
	Token   string `json:"token"`
	Id      string `json:"id"`
	Ver     string `json:"ver"`
	Ability string `json:"ability"`
}

type AppVerUpResult struct {
	Url string `json:"url"`
}

type AppVerDoneTask struct {
	Token string      `json:"token"`
	Id    string      `json:"id"`
	Ver   string      `json:"ver"`
	Info  interface{} `json:"info,omitempty"`
}

type AppMemberAddTask struct {
	Token string `json:"token"`
	Id    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type AppMemberRemoveTask struct {
	Token string `json:"token"`
	Id    string `json:"id"`
	Email string `json:"email"`
}

type AppVerInfoGetTask struct {
	Token string `json:"token"`
	Id    string `json:"id"`
	Ver   string `json:"ver"`
}

type AppApproveTask struct {
	Token       string `json:"token"`
	Id          string `json:"id"`
	ContainerId string `json:"containerId"`
}

type AppUnapproveTask struct {
	Token       string `json:"token"`
	Id          string `json:"id"`
	ContainerId string `json:"containerId"`
}
