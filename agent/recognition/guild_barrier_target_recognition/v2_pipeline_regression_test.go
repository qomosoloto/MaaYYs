package guild_barrier_target_recognition

import (
	"encoding/json"
	"os"
	"testing"

	maa "github.com/MaaXYZ/maa-framework-go/v4"
)

func TestV2PipelinePairsPopupNameWithItsOwnButton(t *testing.T) {
	data, err := os.ReadFile("../../../resource_pack/base/pipeline/战斗/寮突V2.json")
	if err != nil {
		t.Fatal(err)
	}
	var pipeline map[string]struct {
		ROI      maa.Rect        `json:"roi"`
		Expected string          `json:"expected"`
		Params   json.RawMessage `json:"custom_recognition_param"`
	}
	if err := json.Unmarshal(data, &pipeline); err != nil {
		t.Fatal(err)
	}
	type ocrText struct {
		text string
		box  maa.Rect
	}
	// 左弹窗按钮中心为 659，旧版 X=640 分界会错误配到右侧背景玩家。
	leftName := ocrText{"实际左目标", maa.Rect{510, 158, 180, 34}}
	leftButton := ocrText{"进攻", maa.Rect{625, 357, 69, 44}}
	rightName := ocrText{"右侧背景玩家", maa.Rect{850, 158, 180, 34}}
	rightButton := ocrText{"进攻", maa.Rect{945, 357, 69, 44}}
	lowerLeftName := ocrText{"下方左目标", maa.Rect{510, 294, 180, 34}}
	lowerLeftButton := ocrText{"进攻", maa.Rect{626, 495, 68, 40}}
	lowerRightName := ocrText{"下方右目标", maa.Rect{850, 294, 180, 34}}
	lowerRightButton := ocrText{"进攻", maa.Rect{946, 495, 68, 40}}
	bottomName := ocrText{"最下方目标", maa.Rect{510, 430, 180, 34}}
	bottomButton := ocrText{"进攻", maa.Rect{626, 631, 68, 40}}
	tests := []struct {
		name  string
		texts []ocrText
		want  string
		box   maa.Rect
	}{
		{"left_with_readable_background", []ocrText{leftName, leftButton, rightName}, leftName.text, leftButton.box},
		{"left_with_unreadable_background", []ocrText{leftName, leftButton}, leftName.text, leftButton.box},
		{"right_with_readable_background", []ocrText{leftName, rightName, rightButton}, rightName.text, rightButton.box},
		{"lower_left_with_background_names", []ocrText{leftName, rightName, lowerRightName, lowerLeftName, lowerLeftButton}, lowerLeftName.text, lowerLeftButton.box},
		{"lower_right_with_background_names", []ocrText{leftName, rightName, lowerLeftName, lowerRightName, lowerRightButton}, lowerRightName.text, lowerRightButton.box},
		{"bottom_left", []ocrText{leftName, lowerLeftName, bottomName, bottomButton}, bottomName.text, bottomButton.box},
		{"lower_button_with_only_upper_name", []ocrText{leftName, lowerLeftButton}, "", maa.Rect{}},
		{"list_without_popup", []ocrText{leftName, rightName}, "", maa.Rect{}},
		{"button_without_matching_name", []ocrText{rightName, leftButton}, "", maa.Rect{}},
	}
	for _, entry := range []string{"寮突V2-开始观察当前目标", "寮突V2-同一目标持续停留"} {
		params, err := parseParams(&maa.CustomRecognitionArg{CustomRecognitionParam: string(pipeline[entry].Params)})
		if err != nil {
			t.Fatal(err)
		}
		for _, test := range tests {
			t.Run(entry+"/"+test.name, func(t *testing.T) {
				lookup := func(name string, roiOverride *maa.Rect) (*maa.RecognitionDetail, error) {
					node, exists := pipeline[name]
					if !exists {
						t.Fatalf("missing recognition node %s", name)
					}
					for _, text := range test.texts {
						roi, box := node.ROI, text.box
						if roiOverride != nil {
							roi = *roiOverride
						}
						if box.X() < roi.X() || box.Y() < roi.Y() ||
							box.X()+box.Width() > roi.X()+roi.Width() ||
							box.Y()+box.Height() > roi.Y()+roi.Height() ||
							(node.Expected != "" && node.Expected != text.text) {
							continue
						}
						detail, err := json.Marshal(map[string]any{"best": map[string]string{"text": text.text}})
						if err != nil {
							t.Fatal(err)
						}
						return &maa.RecognitionDetail{Hit: true, Box: box, DetailJson: string(detail)}, nil
					}
					return &maa.RecognitionDetail{Hit: false}, nil
				}
				name, box, ok := recognizeCurrentTargetWith(lookup, params.TargetLayouts...)
				if name != test.want || box != test.box || ok != (test.want != "") {
					t.Fatalf("target = %q, box = %v, hit = %v; want %q, %v", name, box, ok, test.want, test.box)
				}
			})
		}
	}
}
