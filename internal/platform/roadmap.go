package platform

type Feature struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Summary      string   `json:"summary"`
	Value        string   `json:"value"`
	Phase        string   `json:"phase"`
	Status       string   `json:"status"`
	Dependencies []string `json:"dependencies"`
	Deliverables []string `json:"deliverables"`
}

type Roadmap struct {
	Product     string    `json:"product"`
	Vision      string    `json:"vision"`
	Recommended []Feature `json:"recommended"`
}

func DefaultRoadmap() Roadmap {
	return Roadmap{
		Product: "buildwebs233",
		Vision:  "构建可扩展的 Go 建站平台，覆盖多站点、低代码、模板、发布、备案和增长分析全链路。",
		Recommended: []Feature{
			{
				ID:      "multi-site-workspace",
				Name:    "多站点与租户工作台",
				Summary: "把当前单页面存储升级为站点中心，支持多站点、多域名、多地区配置。",
				Value:   "这是所有扩展能力的根基，没有 Site/Tenant 维度，后续模板、发布、备案都会很难扩。",
				Phase:   "P0",
				Status:  "foundation",
				Dependencies: []string{
					"site-model",
					"admin-api",
				},
				Deliverables: []string{
					"sites.json 或数据库表",
					"站点列表/创建/编辑 API",
					"区域、域名、模板、状态基础字段",
				},
			},
			{
				ID:      "drag-dsl-builder",
				Name:    "低代码拖拽 DSL 引擎",
				Summary: "把页面从自由文本升级为组件 DSL，支持区块、布局、样式和数据绑定。",
				Value:   "这是拖拽建站的内核，后面模板市场、预览发布、版本回滚都依赖这套结构。",
				Phase:   "P0",
				Status:  "foundation",
				Dependencies: []string{
					"page-schema",
					"component-runtime",
				},
				Deliverables: []string{
					"Block/Section/Layout schema",
					"组件属性模型",
					"预览渲染器",
				},
			},
			{
				ID:      "template-market",
				Name:    "模板市场与行业组件包",
				Summary: "内置企业官网、SaaS、门店、电商活动页等模板，并支持模板复制与升级。",
				Value:   "让产品具备真正的快速建站能力，也是商业化最直接的入口。",
				Phase:   "P0",
				Status:  "foundation",
				Dependencies: []string{
					"drag-dsl-builder",
					"asset-library",
				},
				Deliverables: []string{
					"模板目录与分类",
					"模板预览图与元数据",
					"模板一键套用",
				},
			},
			{
				ID:      "version-review",
				Name:    "页面版本、审批与一键回滚",
				Summary: "每次发布前保留版本快照，支持审稿、审批和快速回退。",
				Value:   "降低误操作风险，让企业团队可以在协作场景中放心使用。",
				Phase:   "P1",
				Status:  "recommended",
				Dependencies: []string{
					"multi-site-workspace",
					"drag-dsl-builder",
				},
				Deliverables: []string{
					"revision snapshot",
					"审批状态流转",
					"一键回滚接口",
				},
			},
			{
				ID:      "publish-pipeline",
				Name:    "发布流水线与环境管理",
				Summary: "支持草稿、预览、正式环境，以及静态导出和 CDN 部署。",
				Value:   "把编辑器与线上交付打通，适合真正的生产站点。",
				Phase:   "P1",
				Status:  "recommended",
				Dependencies: []string{
					"version-review",
					"multi-site-workspace",
				},
				Deliverables: []string{
					"draft/preview/production",
					"静态导出任务",
					"域名与发布历史",
				},
			},
			{
				ID:      "compliance-center",
				Name:    "备案与地区合规中心",
				Summary: "围绕中国 ICP/公安备案与其他地区合规材料，提供资料收集、状态追踪和审核节点。",
				Value:   "这是你目标里最差异化的能力，能把建站工具做成企业可落地平台。",
				Phase:   "P1",
				Status:  "recommended",
				Dependencies: []string{
					"multi-site-workspace",
					"publish-pipeline",
				},
				Deliverables: []string{
					"备案字段模型",
					"材料清单与状态",
					"地区合规 checklist",
				},
			},
			{
				ID:      "asset-library",
				Name:    "媒体资源库与 CDN 分发",
				Summary: "统一管理图片、视频、图标和模板截图，并接入 CDN/OSS。",
				Value:   "没有资源库，模板与页面扩展很快会变乱，性能也不好。",
				Phase:   "P1",
				Status:  "recommended",
				Dependencies: []string{
					"multi-site-workspace",
				},
				Deliverables: []string{
					"资源上传与分类",
					"站点复用资源",
					"静态资源分发策略",
				},
			},
			{
				ID:      "forms-crm",
				Name:    "表单、线索与自动化通知",
				Summary: "为企业官网提供留言、预约、报名、招聘投递等表单能力。",
				Value:   "建站工具只有展示不够，线索采集才是真正的业务闭环。",
				Phase:   "P2",
				Status:  "recommended",
				Dependencies: []string{
					"drag-dsl-builder",
					"multi-site-workspace",
				},
				Deliverables: []string{
					"表单组件",
					"线索管理台",
					"邮件/Webhook 通知",
				},
			},
			{
				ID:      "seo-performance",
				Name:    "SEO、站点性能与可访问性中心",
				Summary: "提供 sitemap、robots、Meta、结构化数据和性能检查。",
				Value:   "企业站最终要能被搜到、打开快、满足基础可访问性。",
				Phase:   "P2",
				Status:  "recommended",
				Dependencies: []string{
					"publish-pipeline",
				},
				Deliverables: []string{
					"SEO 设置页",
					"性能检查建议",
					"可访问性报告",
				},
			},
			{
				ID:      "analytics-extensibility",
				Name:    "分析看板与插件扩展平台",
				Summary: "接入事件分析、来源分析、表单转化，并保留插件机制给未来扩展。",
				Value:   "这能把建站平台升级成增长平台，也让你未来有生态空间。",
				Phase:   "P2",
				Status:  "recommended",
				Dependencies: []string{
					"forms-crm",
					"publish-pipeline",
				},
				Deliverables: []string{
					"站点分析 API",
					"转化看板",
					"插件注册机制",
				},
			},
		},
	}
}
