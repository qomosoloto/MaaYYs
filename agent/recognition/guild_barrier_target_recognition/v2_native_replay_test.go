package guild_barrier_target_recognition

import (
	"encoding/json"
	"image"
	"image/draw"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
)

// 显式提供本地截图时启用；只回放识别，不连接设备或发送输入。
func TestV2NativeScreenshotReplay(t *testing.T) {
	encodedFixtures := os.Getenv("MAAYYS_GUILD_REPLAY")
	if encodedFixtures == "" {
		t.Skip("set MAAYYS_GUILD_REPLAY and MAAYYS_NATIVE_LIB_DIR for native screenshot replay")
	}
	var fixtures []struct {
		Path       string `json:"path"`
		Target     string `json:"target"`
		ButtonYMin int    `json:"button_y_min"`
		ButtonYMax int    `json:"button_y_max"`
	}
	if err := json.Unmarshal([]byte(encodedFixtures), &fixtures); err != nil || len(fixtures) == 0 {
		t.Fatalf("invalid replay fixtures: %v", err)
	}
	if err := maa.Init(maa.WithLibDir(os.Getenv("MAAYYS_NATIVE_LIB_DIR"))); err != nil {
		t.Fatal(err)
	}
	defer maa.Release()
	if err := maa.ConfigInitOption(t.TempDir(), "{}"); err != nil {
		t.Fatal(err)
	}
	resource, err := maa.NewResource()
	if err != nil {
		t.Fatal(err)
	}
	defer resource.Destroy()
	for _, bundle := range []string{"base", "official", "huawei"} {
		path, err := filepath.Abs("../../../resource_pack/" + bundle)
		if err != nil || !resource.PostBundle(path).Wait().Success() {
			t.Fatalf("load resource %s: %v", bundle, err)
		}
	}
	recognizer := &GuildBarrierTargetRecognition{logf: func(string, ...any) {}}
	if err := resource.RegisterCustomRecognition("GuildBarrierTargetRecognition", recognizer); err != nil {
		t.Fatal(err)
	}
	tasker, err := maa.NewTasker()
	if err != nil {
		t.Fatal(err)
	}
	defer tasker.Destroy()
	if err := tasker.BindResource(resource); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../../resource_pack/base/pipeline/战斗/寮突V2.json")
	if err != nil {
		t.Fatal(err)
	}
	var pipeline map[string]struct {
		Params json.RawMessage `json:"custom_recognition_param"`
	}
	if err := json.Unmarshal(data, &pipeline); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures {
		file, err := os.Open(fixture.Path)
		if err != nil {
			t.Fatal(err)
		}
		screenshot, _, err := image.Decode(file)
		file.Close()
		if err != nil {
			t.Fatal(err)
		}
		// 用户桌面截图先归一到游戏 Pipeline 的 1280x720 基准。
		normalized := image.NewRGBA(image.Rect(0, 0, 1280, 720))
		bounds := screenshot.Bounds()
		for y := 0; y < 720; y++ {
			for x := 0; x < 1280; x++ {
				normalized.Set(x, y, screenshot.At(bounds.Min.X+x*bounds.Dx()/1280, bounds.Min.Y+y*bounds.Dy()/720))
			}
		}
		for _, shift := range []int{0, 320} {
			t.Run(filepath.Base(fixture.Path)+"/"+map[int]string{0: "left", 320: "translated_right"}[shift], func(t *testing.T) {
				img := image.NewRGBA(normalized.Bounds())
				draw.Draw(img, image.Rect(shift, 0, 1280, 720), normalized, image.Point{}, draw.Src)
				job := tasker.PostRecognition(maa.RecognitionTypeCustom, &maa.CustomRecognitionParam{
					CustomRecognition:      "GuildBarrierTargetRecognition",
					CustomRecognitionParam: pipeline["寮突V2-开始观察当前目标"].Params,
				}, img).Wait()
				if !job.Success() {
					t.Fatalf("native recognition failed: %v", job.Error())
				}
				detail, err := job.GetDetail()
				if err != nil || len(detail.NodeDetails) != 1 {
					t.Fatalf("native task detail: %v, %v", detail, err)
				}
				reco := detail.NodeDetails[0].Recognition
				if reco == nil || !reco.Hit || reco.Box.Y() < fixture.ButtonYMin || reco.Box.Y() > fixture.ButtonYMax ||
					reco.Box.X() < 600+shift || reco.Box.X() > 700+shift {
					t.Fatalf("native popup/button pair mismatch: %+v", reco)
				}
				if got := recognizer.consumeAttack(detail.ID, time.Now()); got != fixture.Target {
					t.Fatalf("native target = %q, want %q", got, fixture.Target)
				}
				t.Logf("native paired button=%v", reco.Box)
			})
		}
	}
}
