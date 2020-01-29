package dispatchsim

type RecieveFormat2 struct {
	Command       int `json:"command"`
	SecondCommand int `json:"second_command"`
	Data          interface{}
}
type SettingsFormat struct {
	TaskValueType string `json:"task_value_type"`
}

type ParamaterFormat struct {
	TaskValueType string `json:"task_value_type"`
}
