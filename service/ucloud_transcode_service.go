package service

import (
	"context"
	"fmt"

	"github.com/ucloud/ucloud-sdk-go/services/umedia"
	"github.com/ucloud/ucloud-sdk-go/ucloud"
	"github.com/ucloud/ucloud-sdk-go/ucloud/auth"
	"github.com/ucloud/ucloud-sdk-go/ucloud/config"
)

// UCloudCredentials 包含必要的凭证和配置。
// 请将这些替换为您的实际 UCloud 凭证和设置。
const (
	uCloudPublicKey  = "YOUR_UCLOUD_PUBLIC_KEY"
	uCloudPrivateKey = "YOUR_UCLOUD_PRIVATE_KEY"
	uCloudRegion     = "cn-bj2"     // 例如: "cn-bj2", "us-ca"
	uCloudProjectID  = "org-xxxxxx" // 您的 UCloud 项目 ID
)

// newUMediaClient 创建并返回一个新的 UMedia 客户端。
func newUMediaClient() (*umedia.UMediaClient, error) {
	if uCloudPublicKey == "YOUR_UCLOUD_PUBLIC_KEY" || uCloudPrivateKey == "YOUR_UCLOUD_PRIVATE_KEY" {
		return nil, fmt.Errorf("UCloud 凭证未配置。请设置 uCloudPublicKey 和 uCloudPrivateKey")
	}

	cfg := config.NewConfig()
	cfg.Region = uCloudRegion       // 已修正：直接分配字符串
	cfg.ProjectId = uCloudProjectID // 已修正：直接分配字符串

	cred := auth.NewCredential()
	cred.PublicKey = uCloudPublicKey
	cred.PrivateKey = uCloudPrivateKey

	client := umedia.NewClient(&cfg, &cred)
	return client, nil
}

// CreateTranscodingTemplateParams 定义了创建转码模板的参数。
type CreateTranscodingTemplateParams struct {
	PattenName string // 模板名称 (必填)
	DestFormat string // 目标视频格式 (必填), 例如："mp4", "flv", "mpegts"

	DestVideoBitrate    int    // 视频码率 (必填), 单位 kbps
	DestVideoResolution string // 可选：视频分辨率, 例如："1280x720"
	DestVideoCodec      string // 可选：视频的编码类型, 例如："H264", "H265"

	DestAudioBitrate string // 可选：音频码率, 单位 kbps, 例如："48" (SDK 期望字符串)
	DestAudioSample  int    // 可选：音频采样率, 单位 Hz, 例如：44100
	DestAudioChannel int    // 可选：音频声道数量, 例如：1 (单声道), 2 (双声道)

	// 以下是 SDK 中存在但当前参数结构体未包含的可选字段，可按需添加
	// CallbackUrl       string // 可选: 转码任务结束后，回调客户的url地址
	// DestSuffix        string // 可选: 目标视频的文件名后缀
	// ProjectId         string // 可选: 项目ID
}

// CreateTranscodingTemplate 在 UCloud UMedia 上创建一个新的转码模板。
// 成功时返回 PattenId (模板 ID)。
func CreateTranscodingTemplate(ctx context.Context, params CreateTranscodingTemplateParams) (string, error) {
	client, err := newUMediaClient()
	if err != nil {
		return "", fmt.Errorf("创建 UMedia 客户端失败: %w", err)
	}

	req := client.NewCreateCodecPattenRequest()
	req.PattenName = ucloud.String(params.PattenName)
	req.DestFormat = ucloud.String(params.DestFormat)          // 使用 SDK 定义的 DestFormat
	req.DestVideoBitrate = ucloud.Int(params.DestVideoBitrate) // 使用 SDK 定义的 DestVideoBitrate

	// 可选参数
	if params.DestVideoResolution != "" {
		req.DestVideoResolution = ucloud.String(params.DestVideoResolution)
	}
	if params.DestVideoCodec != "" {
		req.DestVideoCodec = ucloud.String(params.DestVideoCodec)
	}
	if params.DestAudioBitrate != "" { // SDK 期望 *string
		req.DestAudioBitrate = ucloud.String(params.DestAudioBitrate)
	}
	if params.DestAudioSample > 0 { // SDK 期望 *int
		req.DestAudioSample = ucloud.Int(params.DestAudioSample)
	}
	if params.DestAudioChannel > 0 { // SDK 期望 *int. 例如：1 (单声道), 2 (双声道)
		req.DestAudioChannel = ucloud.Int(params.DestAudioChannel)
	}

	// 注意：移除了 PattenType, OutFormat (由DestFormat替代), VideoFrameRate, AudioCodec 等字段
	// 因为它们在您提供的 CreateCodecPattenRequest SDK v0.22.40 定义中不存在或名称已更改。

	resp, err := client.CreateCodecPatten(req)
	if err != nil {
		return "", fmt.Errorf("CreateCodecPatten API 调用失败: %w", err)
	}

	if resp.RetCode != 0 {
		// 已修正：resp.Message 很可能是一个字符串，而不是 *string
		return "", fmt.Errorf("CreateCodecPatten 失败: RetCode=%d, Message=%s", resp.RetCode, resp.Message)
	}

	// 已修正：resp.PattenId 很可能是一个字符串，而不是 *string
	return resp.PattenId, nil
}

// CreateTranscodingTaskParams 定义了创建转码任务的参数。
type CreateTranscodingTaskParams struct {
	SrcURLs           []string // 源视频 URL 列表 (必填，支持最多10条)
	DestBucket        string   // 目标 UFile 存储桶 (必填，bucket全名)
	PattenIDs         []string // 要使用的转码模板 ID 列表 (必填，支持最多3个)
	BaseDir           string   // 可选: 上传到ufile上文件的路径 (输出目录或前缀)
	HeadTailPattenId  string   // 可选: 片头片尾模版Id
	WatermarkPattenId string   // 可选: 水印模版Id
	ProjectId         string   // 可选: 项目ID
}

// CreateTranscodingTaskWithTemplate 使用指定的模板创建一个新的转码任务。
// 成功时返回 TaskId。
func CreateTranscodingTaskWithTemplate(ctx context.Context, params CreateTranscodingTaskParams) ([]ucloud.TaskIdLis, error) {
	client, err := newUMediaClient()
	if err != nil {
		return nil, fmt.Errorf("创建 UMedia 客户端失败: %w", err)
	}

	req := client.NewCreateCodecTaskByPattenRequest() // 已由用户更新

	// 必填字段
	// 直接赋值，如果 req.Url 期望 []string 类型
	req.Url = params.SrcURLs
	req.DestBucket = ucloud.String(params.DestBucket)
	req.CodecPattenId = params.PattenIDs

	// 可选字段
	if params.BaseDir != "" {
		req.BaseDir = ucloud.String(params.BaseDir)
	}
	if params.HeadTailPattenId != "" {
		req.HeadTailPattenId = ucloud.String(params.HeadTailPattenId)
	}
	if params.WatermarkPattenId != "" {
		req.WatermarkPattenId = ucloud.String(params.WatermarkPattenId)
	}
	if params.ProjectId != "" {
		req.ProjectId = ucloud.String(params.ProjectId)
	}

	// 假设执行方法也相应更改为 CreateCodecTaskByPatten
	resp, err := client.CreateCodecTaskByPatten(req) // FIXME: 请验证此方法名称是否正确 for SDK v0.22.40
	if err != nil {
		return "", fmt.Errorf("CreateCodecTaskByPatten API 调用失败: %w", err)
	}

	if resp.RetCode != 0 {
		return "", fmt.Errorf("CreateCodecTaskByPatten 失败: RetCode=%d, Message=%s", resp.RetCode, resp.Message)
	}

	// 假设 TaskId 字段在响应中仍然存在且名称相同
	// FIXME: 请验证响应结构体中 TaskId 字段的名称和类型 for SDK v0.22.40
	return resp.TaskIdList, nil
}

/*
// 用法示例 (您可以将其放在 main 函数或测试中)
func main() {
	ctx := context.Background()

	// 1. 创建一个转码模板
	templateParams := CreateTranscodingTemplateParams{
		PattenName:          "MyMP4TemplateHD",
		DestFormat:          "mp4", // 更新字段名
		DestVideoBitrate:    2000,  // 更新字段名, kbps
		DestVideoResolution: "1280x720", // 更新字段名
		DestVideoCodec:      "H264",     // 更新字段名
		DestAudioBitrate:    "128",      // 更新字段名和类型 (string), kbps
		DestAudioSample:     44100,      // 更新字段名和类型 (int)
		DestAudioChannel:    2,          // 更新字段名和类型 (int), 2 代表 STEREO
		// PattenType, OutFormat, VideoFrameRate, AudioCodec 已移除
	}
	pattenID, err := CreateTranscodingTemplate(ctx, templateParams)
	if err != nil {
		fmt.Printf("创建模板错误: %v\\n", err)
		return
	}
	fmt.Printf("成功创建模板。PattenID: %s\\n", pattenID)

	// 2. 使用模板创建一个转码任务
	if pattenID != "" {
		taskParams := CreateTranscodingTaskParams{
			SrcURLs:    []string{"http://your-source-bucket.ufile.ucloud.cn/input.mp4"}, // 更新为列表
			DestBucket: "your-destination-bucket",                                      // 替换为您的目标 UFile 存储桶
			PattenIDs:  []string{pattenID},                                             // 更新为列表
			BaseDir:    "output/",                                                      // 可选：输出目录前缀，替换为您期望的输出路径
			// HeadTailPattenId: "htpatten-xxxx", // 可选
			// WatermarkPattenId: "wmpatten-xxxx", // 可选
		}
		taskID, err := CreateTranscodingTaskWithTemplate(ctx, taskParams)
		if err != nil {
			fmt.Printf("创建转码任务错误: %v\n", err)
			return
		}
		fmt.Printf("成功创建转码任务。TaskID: %s\n", taskID)
	}
}
*/
