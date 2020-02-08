package generator

import (
	"fmt"
	"go/format"
	"regexp"
	"strings"

	"github.com/gdexlab/go-render/render"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/PrasadG193/kyaml2go/pkg/importer"
	"github.com/PrasadG193/kyaml2go/pkg/kube"
)

// KubeMethod define methods to manage K8s resource
type KubeMethod string

const (
	// MethodCreate to create K8s resource
	MethodCreate = "create"
	// MethodGet to get K8s resource
	MethodGet    = "get"
	// MethodUpdate to update K8s resource
	MethodUpdate = "update"
	// MethodDelete to delete K8s resource
	MethodDelete = "delete"
)

// CodeGen holds K8s resource object
type CodeGen struct {
	raw             []byte
	method          KubeMethod
	name            string
	namespace       string
	kind            string
	group           string
	version         string
	replicaCount    string
	termGracePeriod string
	imports         string
	kubeClient      string
	runtimeObject   runtime.Object
	kubeObject      string
	kubeManage      string
	extraFuncs      map[string]string
}

func (m KubeMethod) String() string {
	return string(m)
}

// New returns instance of CodeGen
func New(raw []byte, method KubeMethod) CodeGen {
	return CodeGen{
		raw:        raw,
		method:     method,
		extraFuncs: make(map[string]string),
	}
}

// Generate returns Go code for KubeMethod on a K8s resource
func (c *CodeGen) Generate() (code string, err error) {
	// Convert yaml specs to runtime object
	if err = c.addKubeObject(); err != nil {
		return code, err
	}

	// Create kubeclient
	c.addKubeClient()
	// Add methods to kubeclient
	c.addKubeManage()
	// Remove unnecessary fields
	c.cleanupObject()

	if c.method != MethodDelete && c.method != MethodGet {
		i := importer.New(c.kind, c.group, c.version, c.kubeObject)
		c.imports, c.kubeObject = i.FindImports()
		c.addPtrMethods()
	}

	return c.prettyCode()
}

// addKubeObject converts raw yaml specs to runtime object
func (c *CodeGen) addKubeObject() error {
	var err error
	var objMeta *schema.GroupVersionKind
	decode := scheme.Codecs.UniversalDeserializer().Decode
	c.runtimeObject, objMeta, err = decode(c.raw, nil, nil)
	if err != nil || objMeta == nil {
		return fmt.Errorf("Error while decoding YAML object. Err was: %s", err)
	}

	// Find group, kind and version
	c.kind = strings.Title(objMeta.Kind)
	c.group = strings.Title(objMeta.Group)
	if len(c.group) == 0 {
		c.group = "Core"
	}
	c.version = strings.Title(objMeta.Version)

	// Find replica count
	var re = regexp.MustCompile(`replicas:\s?([0-9]+)`)
	matched := re.FindAllStringSubmatch(string(c.raw), -1)
	if len(matched) == 1 && len(matched[0]) == 2 {
		c.replicaCount = matched[0][1]
	}

	// Add terminationGracePeriodSeconds
	re = regexp.MustCompile(`terminationGracePeriodSeconds:\s?([0-9]+)`)
	matched = re.FindAllStringSubmatch(string(c.raw), -1)
	if len(matched) == 1 && len(matched[0]) == 2 {
		c.termGracePeriod = matched[0][1]
	}

	// Add object name
	re = regexp.MustCompile(`name:\s?"?([-a-zA-Z\.]+)`)
	matched = re.FindAllStringSubmatch(string(c.raw), -1)
	if len(matched) >= 1 && len(matched[0]) == 2 {
		c.name = matched[0][1]
	}

	// Add namespace
	c.namespace = "default"
	re = regexp.MustCompile(`namespace:\s?"?([-a-zA-Z]+)`)
	matched = re.FindAllStringSubmatch(string(c.raw), -1)
	if len(matched) >= 1 && len(matched[0]) == 2 {
		c.namespace = matched[0][1]
	}

	// Replace Data with StringData for secret object types
	c.secretStringData()

	// Pretty struct
	c.kubeObject = prettyStruct(render.AsCode(c.runtimeObject))
	return nil
}

// addKubeClient adds code to create kube client
func (c *CodeGen) addKubeClient() {
	c.kubeClient = fmt.Sprintf(`var kubeconfig string
	kubeconfig, ok := os.LookupEnv("KUBECONFIG")
	if !ok {
		kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

        config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
        if err != nil {
                panic(err)
        }
        clientset, err := kubernetes.NewForConfig(config)
        if err != nil {
                panic(err)
        }
	`)

	// Pod => Pods
	kindPlurals := fmt.Sprintf("%ss", c.kind)
	// Ingress => Ingresses
	if strings.HasSuffix(c.kind, "ss") {
		kindPlurals = fmt.Sprintf("%ses", c.kind)
	}
	// PodSecurityPolicy => PodSecurityPolicies
	if strings.HasSuffix(c.kind, "y") {
		// Ingress => Ingresses
		kindPlurals = fmt.Sprintf("%sies", strings.TrimRight(c.kind, "y"))
	}

	method := fmt.Sprintf("kubeclient := clientset.%s%s().%s()", strings.Split(c.group, ".")[0], c.version, kindPlurals)
	if _, ok := kube.KindNamespaced[c.kind]; ok {
		method = fmt.Sprintf("kubeclient := clientset.%s%s().%s(\"%s\")", strings.Split(c.group, ".")[0], c.version, kindPlurals, c.namespace)
	}
	c.kubeClient += method
}

// addKubeManage add methods to manage job resource
func (c *CodeGen) addKubeManage() {
	var method string
	switch c.method {
	case MethodDelete:
		// Add imports
		for _, i := range importer.CommonImports {
			c.imports += fmt.Sprintf("\"%s\"\n", i)
		}
		c.imports += "metav1 \"k8s.io/apimachinery/pkg/apis/meta/v1\"\n"

		param := fmt.Sprintf(`"%s", &metav1.DeleteOptions{}`, c.name)
		method = fmt.Sprintf("err = kubeclient.%s(%s)", strings.Title(c.method.String()), param)

	case MethodGet:
		// Add imports
		for _, i := range importer.CommonImports {
			c.imports += fmt.Sprintf("\"%s\"\n", i)
		}
		c.imports += "metav1 \"k8s.io/apimachinery/pkg/apis/meta/v1\"\n"

		param := fmt.Sprintf(`"%s", metav1.GetOptions{}`, c.name)
		method = fmt.Sprintf("found, err := kubeclient.%s(%s)\n", strings.Title(c.method.String()), param)
		// Add log
		method += fmt.Sprintf(`fmt.Printf("Found object : %s", found)`, "%+v")

	default:
		method = fmt.Sprintf("_, err = kubeclient.%s(object)", strings.Title(c.method.String()))
	}

	c.kubeManage = fmt.Sprintf(`%s
        if err != nil {
                panic(err)
        }
	`, method)

	if c.method != MethodGet {
		c.kubeManage += fmt.Sprintf(`fmt.Println("%s %sd successfully!")`, c.kind, strings.Title(c.method.String()))
	}
}

// prettyCode generates final go code well indented by gofmt
func (c *CodeGen) prettyCode() (code string, err error) {
	kubeobject := fmt.Sprintf(`// Create resource object
	object := %s`, c.kubeObject)

	if c.method == MethodDelete || c.method == MethodGet {
		kubeobject = ""
	}

	main := fmt.Sprintf(`
	// Auto-generated by kyaml2go - https://github.com/PrasadG193/kyaml2go
	package main

	import (
		%s
	)

	func main() {
	// Create client
	%s

	%s

	// Manage resource
	%s
	}
	`, c.imports, c.kubeClient, kubeobject, c.kubeManage)

	// Add pointer methods
	for _, f := range c.extraFuncs {
		main += f
	}

	// Run gofmt
	goFormat, err := format.Source([]byte(main))
	if err != nil {
		return code, fmt.Errorf("go fmt error: %s", err.Error())
	}
	return string(goFormat), nil
}

// cleanupObject removes fields with nil values
func (c *CodeGen) cleanupObject() {
	if c.method == MethodDelete || c.method == MethodGet {
		c.kubeObject = ""
	}
	kubeObject := strings.Split(c.kubeObject, "\n")
	kubeObject = replaceSubObject(kubeObject, "CreationTimestamp", "", -1)
	kubeObject = replaceSubObject(kubeObject, "Status", "", -1)
	kubeObject = replaceSubObject(kubeObject, "Generation", "", -1)
	kubeObject = removeNilFields(kubeObject)
	kubeObject = updateResources(kubeObject)

	// Remove binary secret data
	if c.kind == "Secret" {
		kubeObject = replaceSubObject(kubeObject, "CreationTimestamp", "", -1)
		kubeObject = replaceSubObject(kubeObject, "Data: map[string][]uint8", "", -1)
	}

	c.kubeObject = ""
	for _, l := range kubeObject {
		if len(l) != 0 {
			c.kubeObject += l + "\n"
		}
	}
}

// secretStringData replaces binary data in resource object to readable string data
func (c *CodeGen) secretStringData() {
	if c.kind != "Secret" {
		return
	}

	secretObject, ok := c.runtimeObject.(*v1.Secret)
	if !ok {
		return
	}
	secretObject.StringData = make(map[string]string)
	for key, val := range secretObject.Data {
		secretObject.StringData[key] = string(val)
	}
	c.runtimeObject = secretObject
}

func prettyStruct(obj string) string {
	obj = strings.Replace(obj, ", ", ",\n", -1)
	obj = strings.Replace(obj, "{", " {\n", -1)
	obj = strings.Replace(obj, "}", ",\n}", -1)

	// Run gofmt
	goFormat, err := format.Source([]byte(obj))
	if err != nil {
		fmt.Println("gofmt error", err)
	}
	return string(goFormat)
}

func removeNilFields(kubeobject []string) []string {
	nilFields := []string{"nil", "\"\"", "false", "{}"}
	for i, line := range kubeobject {
		for _, n := range nilFields {
			if strings.Contains(line, n) {
				kubeobject[i] = ""
			}
		}
	}
	return kubeobject
}

// replace struct field and all sub fields
// n stands for no. of occurances you want to replace
// n < 0 = all occurances
func replaceSubObject(object []string, objectName, newObject string, n int) []string {
	depth := 0

	for i, line := range object {
		if n == 0 {
			break
		}
		if !strings.Contains(line, objectName) && depth == 0 {
			continue
		}
		if strings.Contains(line, "{") {
			depth++
		}
		if strings.Contains(line, "}") {
			depth--
		}
		if strings.Contains(line, objectName) {
			object[i] = newObject
		} else {
			object[i] = ""
		}
		// Replace n occurances
		if depth == 0 {
			n--
		}
	}
	return object
}

func parseResourceValue(object []string) (string, string) {
	var value, format string
	for _, line := range object {
		// parse value
		re := regexp.MustCompile(`(?m)value:\s([0-9]*)`)
		matched := re.FindAllStringSubmatch(line, -1)
		if len(matched) >= 1 && len(matched[0]) == 2 {
			value = matched[0][1]
		}

		// Parse unit
		re = regexp.MustCompile(`(?m)resource\.Format\("([a-z-A-Z]*)"\)`)
		matched = re.FindAllStringSubmatch(line, -1)
		if len(matched) >= 1 && len(matched[0]) == 2 {
			format = matched[0][1]
			break
		}
	}
	return value, format
}

func (c *CodeGen) addPtrMethods() {
	object := strings.Split(c.kubeObject, "\n")
	for i, line := range object {
		var typeName, funcName, param string
		re := regexp.MustCompile(`(?m)\(&([a-zA-Z0-9]*.([a-zA-Z]*))\)\(([a-zA-Z0-9-\/"]*)\)`)
		matched := re.FindAllStringSubmatch(line, -1)
		if len(matched) == 1 && len(matched[0]) == 4 {
			typeName = matched[0][1]
			funcName = "ptr" + matched[0][2]
			if len(matched[0][2]) == 0 {
				funcName = "ptr" + matched[0][1]
			}
			param = matched[0][3]
			object[i] = strings.Replace(object[i], matched[0][0], fmt.Sprintf("%s(%s)", funcName, param), 1)
			c.extraFuncs[funcName] = fmt.Sprintf(`
			func %s(p %s) *%s { 
				return &p 
			}
			`, funcName, typeName, typeName)
			// func int%sPtr(i int%s) *int%s { return &i }
		}

		// Fix "&" => "*" values altered by go-render
		re = regexp.MustCompile(`(?m)".*[&].*"`)
		matched = re.FindAllStringSubmatch(line, -1)
		if len(matched) == 1 {
			object[i] = strings.Replace(object[i], "&", "*", -1)
		}
	}
	c.kubeObject = ""
	for _, l := range object {
		if len(l) != 0 {
			c.kubeObject += l + "\n"
		}
	}
}

// e.g "cpu": resource.Quantity(1Gi)" => "cpu": *resource.NewQuantity(700, resource.DecimalSI)"
// TODO: Use resource.MustParse() method instead
func updateResources(object []string) []string {
	resources := []string{"cpu", "memory", "storage", "pods"}
	for i, line := range object {
		for _, res := range resources {
			s := fmt.Sprintf("\"%s\": resource.Quantity", res)
			if strings.Contains(line, s) {
				value, format := parseResourceValue(object[i+1:])
				replaceSubObject(object[i:], s, fmt.Sprintf("\"%s\": *resource.NewQuantity(%s, resource.%s),", res, value, format), 1)
			}
		}
	}
	return object
}
