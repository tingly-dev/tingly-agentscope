package tool

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/mitchellh/mapstructure"
)

type User struct {
	Name string
	Age  int
}

func Handle(u User) {
	fmt.Println("User:", u.Name, u.Age)
}

func TestReflect(t *testing.T) {
	// 模拟输入
	data := map[string]interface{}{
		"Name": "Alice",
		"Age":  18,
	}

	// 👇 获取函数参数类型
	fn := Handle
	fnType := reflect.TypeOf(fn)
	argType := fnType.In(0) // 第一个参数类型

	// 👇 创建这个类型的实例
	argPtr := reflect.New(argType) // *User

	// 👇 map → struct
	err := mapstructure.Decode(data, argPtr.Interface())
	if err != nil {
		panic(err)
	}

	// 👇 转成值（User）
	argVal := argPtr.Elem()

	// 👇 调用函数
	reflect.ValueOf(fn).Call([]reflect.Value{argVal})
}
