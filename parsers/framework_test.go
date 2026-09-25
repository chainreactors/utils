package parsers

import (
	"encoding/json"
	"testing"
)

func TestFrameworksJudgement(t *testing.T) {
	fs := Frameworks{}
	fs.Add(&Framework{Name: "nginx", Judge: &Judgement{Layer: "web_server", Confidence: 0.9}})
	fs.Add(&Framework{Name: "jenkins", Judge: &Judgement{Layer: "application", Primary: true}})
	fs.Add(&Framework{Name: "tomcat", Judge: &Judgement{Rejected: true}})
	fs.Add(&Framework{Name: "apache-tomcat", Judge: &Judgement{Duplicate: true}})
	fs.Add(&Framework{Name: "unjudged"})

	accepted := fs.Accepted()
	if len(accepted) != 3 || accepted["tomcat"] != nil || accepted["apache-tomcat"] != nil || accepted["unjudged"] == nil {
		t.Fatalf("accepted %v", accepted.GetNames())
	}
	if p := fs.Primary(); p == nil || p.Name != "jenkins" {
		t.Fatalf("primary %v", p)
	}

	data, err := json.Marshal(fs["nginx"])
	if err != nil {
		t.Fatal(err)
	}
	var back Framework
	if err := json.Unmarshal(data, &back); err != nil || back.Judge == nil || back.Judge.Layer != "web_server" || back.Judge.Confidence != 0.9 {
		t.Fatalf("round trip %s -> %+v", data, back.Judge)
	}
	if data, _ := json.Marshal(fs["unjudged"]); string(data) != `{"name":"unjudged"}` {
		t.Fatalf("unjudged serializes as %s", data)
	}
}

func TestFrameworksAddKeepsJudgement(t *testing.T) {
	fs := Frameworks{}
	fs.Add(NewFramework("nginx", FrameFromDefault))
	judged := NewFramework("nginx", FrameFromDefault)
	judged.Judge = &Judgement{Layer: "web_server"}
	fs.Add(judged)
	if fs["nginx"].Judge == nil {
		t.Fatal("judgement lost on merge")
	}
}
