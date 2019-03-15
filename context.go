package media

type Context struct {
	Media Media
	Data  []map[interface{}]interface{}
}

func NewContext(media Media, data ...map[interface{}]interface{}) *Context {
	if len(data) == 0 {
		data = []map[interface{}]interface{}{{}}
	}
	return &Context{Media: media, Data: data}
}

func (c *Context) GetOk(key interface{}) (value interface{}, ok bool) {
	for i := len(c.Data); i > 0; i-- {
		if value, ok = c.Data[i-1][key]; ok {
			return
		}
	}
	return
}

func (c *Context) With(data map[interface{}]interface{}) func() {
	c.Data = append(c.Data)
	return func() {
		c.Data = c.Data[0 : len(c.Data)-1]
	}
}
