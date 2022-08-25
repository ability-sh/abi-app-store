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
