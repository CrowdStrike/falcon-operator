package k8s_utils

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"

	"k8s.io/client-go/kubernetes/scheme"
)

func ParseK8sObjects(y string) ([]runtime.Object, error) {
	b := bufio.NewReader(strings.NewReader(y))
	r := yaml.NewYAMLReader(b)

	result := []runtime.Object{}
	for {
		doc, err := r.Read()

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}

		d := scheme.Codecs.UniversalDeserializer()
		obj, _, err := d.Decode(doc, nil, nil)
		if err != nil {
			return nil, err
		}

		result = append(result, obj)
	}
	return result, nil
}
