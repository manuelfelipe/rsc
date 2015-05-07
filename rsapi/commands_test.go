package rsapi_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rightscale/rsc/rsapi"
	"github.com/rightscale/rsc/ss/ssm"
)

var _ = Describe("normalize", func() {
	var payload rsapi.ApiParams
	var name string
	var value interface{}

	var res rsapi.ApiParams
	var resErr error

	JustBeforeEach(func() {
		res, resErr = rsapi.Normalize(payload, name, value)
	})

	BeforeEach(func() {
		payload = rsapi.ApiParams{}
	})

	Describe("with a simple string", func() {
		BeforeEach(func() {
			name = "val"
			value = "foo"
		})
		It("sets the value", func() {
			Ω(resErr).ShouldNot(HaveOccurred())
			Ω(res).Should(Equal(rsapi.ApiParams{"val": "foo"}))
		})
	})

	Describe("with an int", func() {
		BeforeEach(func() {
			name = "val"
			value = 42
		})
		It("sets the value", func() {
			Ω(resErr).ShouldNot(HaveOccurred())
			Ω(res).Should(Equal(rsapi.ApiParams{"val": 42}))
		})
	})

	Describe("with a float", func() {
		BeforeEach(func() {
			name = "val"
			value = 42.5
		})
		It("sets the value", func() {
			Ω(resErr).ShouldNot(HaveOccurred())
			Ω(res).Should(Equal(rsapi.ApiParams{"val": 42.5}))
		})
	})

	Describe("with a bool", func() {
		BeforeEach(func() {
			name = "val"
			value = true
		})
		It("sets the value", func() {
			Ω(resErr).ShouldNot(HaveOccurred())
			Ω(res).Should(Equal(rsapi.ApiParams{"val": true}))
		})
	})

	Describe("with a simple array", func() {
		BeforeEach(func() {
			name = "val[]"
			value = []string{"foo"}
		})
		It("sets the value", func() {
			Ω(resErr).ShouldNot(HaveOccurred())
			Ω(res).Should(Equal(rsapi.ApiParams{"val": []string{"foo"}}))
		})
	})

	Describe("with a simple map", func() {
		BeforeEach(func() {
			name = "val[a]"
			value = "foo"
		})
		It("sets the value", func() {
			Ω(resErr).ShouldNot(HaveOccurred())
			Ω(res).Should(Equal(rsapi.ApiParams{"val": rsapi.ApiParams{"a": "foo"}}))
		})
	})

	Describe("with a map of arrays", func() {
		BeforeEach(func() {
			name = "val[a][]"
			value = []interface{}{"foo"}
		})
		It("sets the value", func() {
			Ω(resErr).ShouldNot(HaveOccurred())
			expected := rsapi.ApiParams{
				"val": rsapi.ApiParams{
					"a": []interface{}{"foo"},
				},
			}
			Ω(res).Should(Equal(expected))
		})
	})

	Describe("with a map of arrays with existing values", func() {
		BeforeEach(func() {
			name = "val[a][]"
			value = []string{"foo", "bar"}
		})
		It("sets the value", func() {
			Ω(resErr).ShouldNot(HaveOccurred())
			expected := rsapi.ApiParams{
				"val": rsapi.ApiParams{
					"a": []string{"foo", "bar"},
				},
			}
			Ω(res).Should(Equal(expected))
		})
	})

	Describe("with an array of maps", func() {
		BeforeEach(func() {
			name = "val[][a]"
			value = "foo"
		})
		It("sets the value", func() {
			Ω(resErr).ShouldNot(HaveOccurred())
			expected := rsapi.ApiParams{
				"val": []interface{}{rsapi.ApiParams{"a": "foo"}},
			}
			Ω(res).Should(Equal(expected))
		})
	})

	Describe("with an array of maps of arrays", func() {
		BeforeEach(func() {
			name = "val[][a][]"
			value = []interface{}{"foo"}
		})
		It("sets the value", func() {
			Ω(resErr).ShouldNot(HaveOccurred())
			expected := rsapi.ApiParams{
				"val": []interface{}{rsapi.ApiParams{"a": []interface{}{"foo"}}},
			}
			Ω(res).Should(Equal(expected))
		})
	})

	Describe("with an array of maps with existing keys", func() {
		BeforeEach(func() {
			name = "val[][b]"
			value = "bar"
			payload = rsapi.ApiParams{
				"val": []interface{}{rsapi.ApiParams{"a": "foo"}},
			}
		})
		It("sets the value", func() {
			Ω(resErr).ShouldNot(HaveOccurred())
			expected := rsapi.ApiParams{
				"val": []interface{}{rsapi.ApiParams{"a": "foo", "b": "bar"}},
			}
			Ω(res).Should(Equal(expected))
		})
	})

	Describe("with an array of maps with existing keys with more than one element", func() {
		BeforeEach(func() {
			name = "val[][b]"
			value = "baz"
			payload = rsapi.ApiParams{
				"val": []interface{}{rsapi.ApiParams{"a": "foo", "b": "bar"}},
			}
		})
		It("sets the value", func() {
			Ω(resErr).ShouldNot(HaveOccurred())
			expected := rsapi.ApiParams{
				"val": []interface{}{
					rsapi.ApiParams{"a": "foo", "b": "bar"},
					rsapi.ApiParams{"b": "baz"},
				},
			}
			Ω(res).Should(Equal(expected))
		})
	})
})

var _ = Describe("ParseCommand", func() {
	var cmd, hrefPrefix string
	var values rsapi.ActionCommands
	var api *rsapi.Api

	var parsed *rsapi.ParsedCommand
	var parseErr error

	BeforeEach(func() {
		values = nil
		ssm, err := ssm.New(42, "", nil, nil, nil)
		Ω(err).ShouldNot(HaveOccurred())
		api = ssm.Api
	})

	JustBeforeEach(func() {
		parsed, parseErr = api.ParseCommand(cmd, hrefPrefix, values)
	})

	Describe("Run command line", func() {
		BeforeEach(func() {
			cmd = "run"
			runCmd := rsapi.ActionCommand{
				Href: "/api/manager/projects/42/executions/54",
				Params: []string{
					"name=Tip CWF",
					"configuration_options[][name]=environment_name",
					"configuration_options[][type]=string",
					"configuration_options[][value]=ss2",
				},
			}
			values = rsapi.ActionCommands{"run": &runCmd}
		})
		It("parses", func() {
			Ω(parseErr).ShouldNot(HaveOccurred())
			Ω(parsed).ShouldNot(BeNil())
			payload := rsapi.ApiParams{
				"name": "Tip CWF",
				"configuration_options": []interface{}{rsapi.ApiParams{
					"name":  "environment_name",
					"type":  "string",
					"value": "ss2",
				}},
			}
			expected := rsapi.ParsedCommand{
				HttpMethod:    "POST",
				Uri:           "/projects/42/executions/54/actions/run",
				QueryParams:   rsapi.ApiParams{},
				PayloadParams: payload,
			}
			Ω(*parsed).Should(Equal(expected))
		})
	})

})