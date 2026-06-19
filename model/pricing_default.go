package model

import (
	"strings"
)

// vendorRule 供应商匹配规则。使用有序切片而非 map，确保匹配顺序稳定
// （map 遍历顺序随机，会导致同一模型名被随机归类到不同供应商）。
type vendorRule struct {
	pattern string
	vendor  string
}

// 简化的供应商映射规则（按优先级排序，靠前的规则优先匹配）。
var defaultVendorRules = []vendorRule{
	{"gpt", "OpenAI"},
	{"dall-e", "OpenAI"},
	{"whisper", "OpenAI"},
	{"o1", "OpenAI"},
	{"o3", "OpenAI"},
	{"claude", "Anthropic"},
	{"gemini", "Google"},
	{"moonshot", "Moonshot"},
	{"kimi", "Moonshot"},
	{"chatglm", "智谱"},
	{"glm-", "智谱"},
	{"qwen", "阿里巴巴"},
	{"deepseek", "DeepSeek"},
	{"abab", "MiniMax"},
	{"ernie", "百度"},
	{"spark", "讯飞"},
	{"hunyuan", "腾讯"},
	{"command", "Cohere"},
	{"@cf/", "Cloudflare"},
	{"360", "360"},
	{"yi", "零一万物"},
	{"jina", "Jina"},
	{"mistral", "Mistral"},
	{"grok", "xAI"},
	{"llama", "Meta"},
	{"doubao", "字节跳动"},
	{"kling", "快手"},
	{"jimeng", "即梦"},
	{"vidu", "Vidu"},
}

// 供应商默认图标映射
var defaultVendorIcons = map[string]string{
	"OpenAI":     "OpenAI",
	"Anthropic":  "Claude.Color",
	"Google":     "Gemini.Color",
	"Moonshot":   "Moonshot",
	"智谱":         "Zhipu.Color",
	"阿里巴巴":       "Qwen.Color",
	"DeepSeek":   "DeepSeek.Color",
	"MiniMax":    "Minimax.Color",
	"百度":         "Wenxin.Color",
	"讯飞":         "Spark.Color",
	"腾讯":         "Hunyuan.Color",
	"Cohere":     "Cohere.Color",
	"Cloudflare": "Cloudflare.Color",
	"360":        "Ai360.Color",
	"零一万物":       "Yi.Color",
	"Jina":       "Jina",
	"Mistral":    "Mistral.Color",
	"xAI":        "XAI",
	"Meta":       "Ollama",
	"字节跳动":       "Doubao.Color",
	"快手":         "Kling.Color",
	"即梦":         "Jimeng.Color",
	"Vidu":       "Vidu",
	"微软":         "AzureAI",
	"Microsoft":  "AzureAI",
	"Azure":      "AzureAI",
}

// initDefaultVendorMapping 简化的默认供应商映射
func initDefaultVendorMapping(metaMap map[string]*Model, vendorMap map[int]*Vendor, enableAbilities []AbilityWithChannel) {
	for _, ability := range enableAbilities {
		modelName := ability.Model
		if _, exists := metaMap[modelName]; exists {
			continue
		}

		// 匹配供应商：先按前缀匹配（更可靠，避免 "gpt-5.3-spark" 因含 "spark"
		// 子串被误判为讯飞），未命中再按子串匹配兜底。规则按定义顺序评估，结果稳定。
		vendorID := 0
		modelLower := strings.ToLower(modelName)
		matchedVendor := ""
		for _, rule := range defaultVendorRules {
			if strings.HasPrefix(modelLower, rule.pattern) {
				matchedVendor = rule.vendor
				break
			}
		}
		if matchedVendor == "" {
			for _, rule := range defaultVendorRules {
				if strings.Contains(modelLower, rule.pattern) {
					matchedVendor = rule.vendor
					break
				}
			}
		}
		if matchedVendor != "" {
			vendorID = getOrCreateVendor(matchedVendor, vendorMap)
		}

		// 创建模型元数据
		metaMap[modelName] = &Model{
			ModelName: modelName,
			VendorID:  vendorID,
			Status:    1,
			NameRule:  NameRuleExact,
		}
	}
}

// 查找或创建供应商
func getOrCreateVendor(vendorName string, vendorMap map[int]*Vendor) int {
	// 查找现有供应商
	for id, vendor := range vendorMap {
		if vendor.Name == vendorName {
			return id
		}
	}

	// 创建新供应商
	newVendor := &Vendor{
		Name:   vendorName,
		Status: 1,
		Icon:   getDefaultVendorIcon(vendorName),
	}

	if err := newVendor.Insert(); err != nil {
		return 0
	}

	vendorMap[newVendor.Id] = newVendor
	return newVendor.Id
}

// 获取供应商默认图标
func getDefaultVendorIcon(vendorName string) string {
	if icon, exists := defaultVendorIcons[vendorName]; exists {
		return icon
	}
	return ""
}
