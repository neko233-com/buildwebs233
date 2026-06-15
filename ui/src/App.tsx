import { useEffect, useMemo, useState } from "react";

type Block = {
  id: string;
  type: "text" | "button" | "hero";
  label?: string;
  content?: string;
  props?: Record<string, string>;
};

type Section = {
  id: string;
  name: string;
  layout: string;
  blocks: Block[];
};

type Page = {
  id: string;
  site_id: string;
  name: string;
  slug: string;
  title: string;
  status: string;
  template_id?: string;
  schema_version?: number;
  sections: Section[];
};

type Site = {
  id: string;
  name: string;
  domain: string;
  region: string;
  template_id: string;
  primary_page_id?: string;
  status: string;
  icp_number?: string;
  psb_number?: string;
  compliance?: {
    company_name?: string;
    contact_name?: string;
    contact_phone?: string;
    icp_status?: string;
    psb_status?: string;
    review_status?: string;
    materials?: ComplianceMaterial[];
    review_history?: ComplianceEvent[];
  };
};

type ComplianceMaterial = {
  id: string;
  type: string;
  file_name: string;
  public_url: string;
  status: string;
};

type ComplianceEvent = {
  id: string;
  action: string;
  actor: string;
  note?: string;
  created_at: string;
};

type RoadmapFeature = {
  id: string;
  name: string;
  summary: string;
  value: string;
  phase: string;
  status: string;
};

type Roadmap = {
  product: string;
  vision: string;
  recommended: RoadmapFeature[];
};

type TemplateOption = {
  id: string;
  name: string;
  description: string;
  category: string;
};

const builtinBlocks: Block[] = [
  { id: "b1", type: "text", label: "主标题", props: { text: "主标题" } },
  { id: "b2", type: "text", label: "正文说明", props: { text: "正文说明" } },
  { id: "b3", type: "button", label: "按钮", props: { label: "立即咨询" } },
  { id: "b4", type: "hero", label: "Hero 区", props: { headline: "这是 Hero 区" } },
];

const templateCatalog: TemplateOption[] = [
  { id: "tpl-hero", name: "企业官网模板", description: "通用企业官网首页，适合公司介绍与备案展示。", category: "通用企业" },
  { id: "tpl-product", name: "产品展示模板", description: "适合 SaaS、工具与服务展示。", category: "通用企业" },
  { id: "tpl-enterprise-corporate", name: "企业形象模板", description: "适合品牌官网、资质展示和企业介绍。", category: "通用企业" },
  { id: "tpl-enterprise-factory", name: "制造工厂模板", description: "适合制造、设备与工厂类公司。", category: "通用企业" },
  { id: "tpl-enterprise-service", name: "专业服务模板", description: "适合法务、咨询、顾问类服务站。", category: "通用企业" },
  { id: "tpl-game-arcade", name: "游戏发行模板", description: "适合游戏官网、版本活动与下载入口。", category: "游戏" },
  { id: "tpl-game-esports", name: "电竞赛事模板", description: "适合电竞战队、赛事专题与直播聚合。", category: "游戏" },
  { id: "tpl-game-indie", name: "独立游戏模板", description: "适合独游工作室与玩家社区引流。", category: "游戏" },
  { id: "tpl-ecommerce-fashion", name: "电商服饰模板", description: "适合服饰、美妆与潮流电商。", category: "电商" },
  { id: "tpl-ecommerce-digital", name: "电商数码模板", description: "适合数码、家电与硬件销售。", category: "电商" },
  { id: "tpl-ecommerce-local", name: "本地商城模板", description: "适合同城零售、团购与配送。", category: "电商" },
  { id: "tpl-blog-personal", name: "个人博客模板", description: "适合个人博客与品牌专栏。", category: "博客" },
  { id: "tpl-blog-tech", name: "技术博客模板", description: "适合教程、日志与技术文章。", category: "博客" },
  { id: "tpl-blog-media", name: "媒体专栏模板", description: "适合媒体评论与订阅内容站。", category: "博客" },
  { id: "tpl-technology-saas", name: "SaaS 科技模板", description: "适合 SaaS 平台与云服务官网。", category: "科技" },
  { id: "tpl-technology-ai", name: "AI 科技模板", description: "适合 AI 产品与自动化服务。", category: "科技" },
  { id: "tpl-technology-devtool", name: "开发工具模板", description: "适合开发者平台与基础设施产品。", category: "科技" },
  { id: "tpl-science-lab", name: "实验室模板", description: "适合实验室、研究机构与课题组。", category: "科研" },
  { id: "tpl-science-research", name: "科研项目模板", description: "适合项目成果与学术合作展示。", category: "科研" },
  { id: "tpl-science-education", name: "科普教育模板", description: "适合科普平台与教育机构。", category: "科研" },
  { id: "tpl-outsourcing-agency", name: "外包机构模板", description: "适合综合外包与商务服务公司。", category: "外包" },
  { id: "tpl-outsourcing-software", name: "软件外包模板", description: "适合软件定制与研发交付团队。", category: "外包" },
  { id: "tpl-outsourcing-design", name: "设计外包模板", description: "适合设计工作室与视觉服务。", category: "外包" },
  { id: "tpl-music-artist", name: "音乐人模板", description: "适合歌手、乐队与作品推广。", category: "音乐" },
  { id: "tpl-music-label", name: "厂牌模板", description: "适合音乐厂牌与版权合作。", category: "音乐" },
  { id: "tpl-music-festival", name: "音乐节模板", description: "适合演出档期与票务说明。", category: "音乐" },
  { id: "tpl-culture-museum", name: "文博馆模板", description: "适合博物馆、美术馆与文博机构。", category: "文化" },
  { id: "tpl-culture-brand", name: "文化品牌模板", description: "适合文创品牌与文化故事站。", category: "文化" },
  { id: "tpl-culture-event", name: "文化活动模板", description: "适合论坛、展演与公共文化项目。", category: "文化" },
  { id: "tpl-news-local", name: "地方资讯模板", description: "适合地方资讯与民生信息站。", category: "新闻" },
  { id: "tpl-news-tech", name: "科技资讯模板", description: "适合科技媒体与行业快讯。", category: "新闻" },
  { id: "tpl-news-finance", name: "财经资讯模板", description: "适合财经媒体与研究简报。", category: "新闻" },
  { id: "tpl-medical-clinic", name: "门诊医疗模板", description: "适合诊所、门诊与专科机构。", category: "医疗" },
  { id: "tpl-medical-hospital", name: "医院机构模板", description: "适合综合医院与医疗集团。", category: "医疗" },
  { id: "tpl-medical-telehealth", name: "互联网医疗模板", description: "适合在线问诊与健康平台。", category: "医疗" },
  { id: "tpl-education-school", name: "学校机构模板", description: "适合学校、培训机构与校区介绍。", category: "教育" },
  { id: "tpl-education-course", name: "课程招生模板", description: "适合课程介绍与在线报名。", category: "教育" },
  { id: "tpl-education-vocational", name: "职业教育模板", description: "适合职业培训与认证机构。", category: "教育" },
  { id: "tpl-finance-bank", name: "金融服务模板", description: "适合银行、理财与保险业务。", category: "金融" },
  { id: "tpl-finance-advisory", name: "投顾咨询模板", description: "适合企业金融与顾问服务。", category: "金融" },
  { id: "tpl-finance-fintech", name: "金融科技模板", description: "适合支付、风控与账务产品。", category: "金融" },
];

const complianceMaterialTypes = [
  { value: "business-license", label: "营业执照" },
  { value: "legal-identity", label: "法人身份证明" },
  { value: "domain-proof", label: "域名持有证明" },
  { value: "hosting-proof", label: "接入/服务器证明" },
  { value: "medical-license", label: "医疗资质" },
  { value: "publication-license", label: "出版/内容资质" },
  { value: "finance-license", label: "金融牌照" },
  { value: "general", label: "其他补充材料" },
];

export default function App() {
  const [pages, setPages] = useState<Page[]>([]);
  const [draft, setDraft] = useState<Block[]>([]);
  const [sites, setSites] = useState<Site[]>([]);
  const [roadmap, setRoadmap] = useState<Roadmap | null>(null);
  const [activeSiteId, setActiveSiteId] = useState("site-default");
  const [selectedTemplateId, setSelectedTemplateId] = useState("tpl-hero");
  const [uploadFile, setUploadFile] = useState<File | null>(null);
  const [materialType, setMaterialType] = useState("business-license");
  const [reviewNote, setReviewNote] = useState("");
  const [dragIndex, setDragIndex] = useState<number | null>(null);
  const [templateConfig, setTemplateConfig] = useState({
    siteName: "",
    headline: "",
    subheadline: "",
    ctaLabel: "",
  });

  const previewHTML = useMemo(() => {
    if (draft.length === 0) {
      return "<p>拖拽左侧区块到右侧画布开始搭建页面</p>";
    }
    return draft
      .map((b) => {
        if (b.type === "button") {
          return `<a class='preview-btn'>${b.props?.label ?? b.label ?? "按钮"}</a>`;
        }
        if (b.type === "hero") {
          return `<div class='preview-hero'>${b.props?.headline ?? b.label ?? "Hero"}</div>`;
        }
        return `<p>${b.props?.text ?? b.label ?? "文本"}</p>`;
      })
      .join("");
  }, [draft]);

  useEffect(() => {
    loadPages().then(setPages).catch(() => setPages([]));
    loadSites().then(setSites).catch(() => setSites([]));
    loadRoadmap().then(setRoadmap).catch(() => setRoadmap(null));
  }, []);

  useEffect(() => {
    loadPages(activeSiteId).then(setPages).catch(() => setPages([]));
  }, [activeSiteId]);

  const onDrop = (block: Block) => {
    setDraft((prev) => [...prev, { ...block, id: `${block.id}-${Date.now()}` }]);
  };

  const reorderDraft = (fromIndex: number, toIndex: number) => {
    if (fromIndex === toIndex || toIndex < 0 || toIndex >= draft.length) {
      return;
    }
    setDraft((prev) => {
      const next = [...prev];
      const [moved] = next.splice(fromIndex, 1);
      next.splice(toIndex, 0, moved);
      return next;
    });
  };

  const saveDraft = async () => {
    if (!draft.length) {
      return;
    }
    const doSave = async () =>
      await fetch("/api/admin/pages", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
    const payload: Page = {
      id: "",
      site_id: activeSiteId,
      name: "新建页面",
      slug: "",
      title: "新建页面",
      status: "draft",
      schema_version: 2,
      sections: [
        {
          id: `section-${Date.now()}`,
          name: "主区域",
          layout: "stack",
          blocks: draft,
        },
      ],
    };
    let res = await doSave();
    if (res.status === 401) {
      await fetch("/api/login", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: "root", password: "root" }),
      });
      res = await doSave();
    }
    if (!res.ok) {
      return;
    }
    setDraft([]);
    await loadPages(activeSiteId).then(setPages);
  };

  const createSiteSkeleton = async () => {
    const payload: Site = {
      id: "",
      name: `备案企业站 ${sites.length + 1}`,
      domain: "",
      region: "CN",
      template_id: "tpl-hero",
      status: "planning",
      compliance: {
        icp_status: "not_started",
        psb_status: "not_started",
        review_status: "draft",
      },
    };

    let res = await fetch("/api/admin/sites", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });
    if (res.status === 401) {
      await fetch("/api/login", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username: "root", password: "root" }),
      });
      res = await fetch("/api/admin/sites", {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
    }
    if (!res.ok) {
      return;
    }
    const nextSites = await loadSites();
    setSites(nextSites);
    setActiveSiteId(nextSites[0]?.id ?? activeSiteId);
  };

  const activeSite = sites.find((site) => site.id === activeSiteId) ?? null;
  const activeSitePages = pages.filter((page) => page.site_id === activeSiteId);
  const selectedTemplate = templateCatalog.find((template) => template.id === selectedTemplateId) ?? templateCatalog[0];
  const templateGroups = useMemo(() => {
    return templateCatalog.reduce<Record<string, TemplateOption[]>>((acc, template) => {
      acc[template.category] = acc[template.category] ?? [];
      acc[template.category].push(template);
      return acc;
    }, {});
  }, []);

  useEffect(() => {
    const primaryPage = activeSitePages.find((page) => page.id === activeSite?.primary_page_id) ?? activeSitePages[0];
    const heroBlock = primaryPage?.sections?.[0]?.blocks?.find((block) => block.type === "hero");
    const textBlock = primaryPage?.sections?.flatMap((section) => section.blocks).find((block) => block.type === "text");
    const buttonBlock = primaryPage?.sections?.flatMap((section) => section.blocks).find((block) => block.type === "button");
    setTemplateConfig({
      siteName: activeSite?.name ?? "",
      headline: heroBlock?.props?.headline ?? "",
      subheadline: textBlock?.props?.text ?? "",
      ctaLabel: buttonBlock?.props?.label ?? "",
    });
  }, [activeSiteId, activeSite?.name, activeSite?.primary_page_id, activeSitePages]);

  useEffect(() => {
    if (activeSite?.template_id) {
      setSelectedTemplateId(activeSite.template_id);
    }
  }, [activeSite?.template_id]);

  const applyTemplate = async () => {
    const res = await authedJsonFetch(`/api/admin/sites/${activeSiteId}/apply-template`, {
      template_id: selectedTemplateId,
    });
    if (!res.ok) {
      return;
    }
    await Promise.all([loadSites().then(setSites), loadPages(activeSiteId).then(setPages)]);
  };

  const publishPrimaryPage = async () => {
    if (!activeSite?.primary_page_id) {
      return;
    }
    const res = await authedPost(`/api/admin/pages/${activeSite.primary_page_id}/publish`);
    if (!res.ok) {
      return;
    }
    await loadPages(activeSiteId).then(setPages);
  };

  const setHomepage = async (pageId: string) => {
    const res = await authedJsonFetch(`/api/admin/sites/${activeSiteId}/homepage`, {
      page_id: pageId,
    });
    if (!res.ok) {
      return;
    }
    await loadSites().then(setSites);
  };

  const submitCompliance = async (action: string) => {
    const res = await authedJsonFetch(`/api/admin/sites/${activeSiteId}/compliance/review`, {
      action,
      note: "",
    });
    if (!res.ok) {
      return;
    }
    await loadSites().then(setSites);
  };

  const uploadComplianceMaterial = async () => {
    if (!uploadFile) {
      return;
    }
    await ensureAuth();
    const form = new FormData();
    form.append("type", materialType);
    form.append("file", uploadFile);
    const res = await fetch(`/api/admin/sites/${activeSiteId}/compliance/materials`, {
      method: "POST",
      credentials: "include",
      body: form,
    });
    if (!res.ok) {
      return;
    }
    setUploadFile(null);
    setReviewNote("");
    await loadSites().then(setSites);
  };

  const reviewMaterial = async (materialId: string) => {
    const res = await authedJsonFetch(`/api/admin/sites/${activeSiteId}/compliance/review`, {
      action: "mark_material_verified",
      note: reviewNote,
      material_id: materialId,
    });
    if (!res.ok) {
      return;
    }
    setReviewNote("");
    await loadSites().then(setSites);
  };

  const saveTemplateConfig = async () => {
    if (!activeSite) {
      return;
    }
    await ensureAuth();
    const siteRes = await fetch("/api/admin/sites", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        ...activeSite,
        name: templateConfig.siteName || activeSite.name,
      }),
    });
    if (!siteRes.ok) {
      return;
    }

    const targetPage =
      activeSitePages.find((page) => page.id === activeSite.primary_page_id) ?? activeSitePages[0];
    if (!targetPage) {
      await loadSites().then(setSites);
      return;
    }
    const nextSections = targetPage.sections.map((section) => ({
      ...section,
      blocks: section.blocks.map((block) => {
        if (block.type === "hero") {
          return {
            ...block,
            props: { ...(block.props ?? {}), headline: templateConfig.headline || templateConfig.siteName },
          };
        }
        if (block.type === "button") {
          return {
            ...block,
            props: { ...(block.props ?? {}), label: templateConfig.ctaLabel || "立即咨询" },
          };
        }
        if (block.type === "text") {
          return {
            ...block,
            props: { ...(block.props ?? {}), text: templateConfig.subheadline || block.props?.text || "" },
          };
        }
        return block;
      }),
    }));

    const pageRes = await fetch("/api/admin/pages", {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        ...targetPage,
        name: templateConfig.siteName ? `${templateConfig.siteName} 首页` : targetPage.name,
        title: templateConfig.siteName || targetPage.title,
        sections: nextSections,
      }),
    });
    if (!pageRes.ok) {
      return;
    }
    await Promise.all([loadSites().then(setSites), loadPages(activeSiteId).then(setPages)]);
  };

  return (
    <main className="shell">
      <section className="palette">
        <h2>组件库</h2>
        {builtinBlocks.map((block) => (
          <button
            key={block.id}
            className="block-item"
            onClick={() => onDrop(block)}
          >
            {block.label}
          </button>
        ))}
      </section>
      <section className="canvas">
        <div className="panel-head">
          <h2>画布</h2>
          <select
            value={activeSiteId}
            onChange={(event) => setActiveSiteId(event.target.value)}
          >
            {sites.map((site) => (
              <option key={site.id} value={site.id}>
                {site.name}
              </option>
            ))}
          </select>
        </div>
        <div className="frame" dangerouslySetInnerHTML={{ __html: previewHTML }} />
        <div className="action-row">
          <button onClick={saveDraft}>保存页面</button>
          <button onClick={publishPrimaryPage}>发布首页</button>
        </div>
        <div className="draft-list">
          {draft.map((block, index) => (
            <div
              key={block.id}
              className="draft-item"
              draggable
              onDragStart={() => setDragIndex(index)}
              onDragOver={(event) => event.preventDefault()}
              onDrop={() => {
                if (dragIndex !== null) {
                  reorderDraft(dragIndex, index);
                }
                setDragIndex(null);
              }}
            >
              <strong>{block.label ?? block.type}</strong>
              <div className="action-row">
                <button onClick={() => reorderDraft(index, index - 1)}>上移</button>
                <button onClick={() => reorderDraft(index, index + 1)}>下移</button>
              </div>
            </div>
          ))}
        </div>
        <p className="muted">当前页面将保存到站点：{activeSite?.name ?? "默认企业站"}</p>
      </section>
      <section className="sites">
        <div className="panel-head">
          <h2>站点中心</h2>
          <button onClick={createSiteSkeleton}>创建站点骨架</button>
        </div>
        <div className="template-bar">
          <select value={selectedTemplateId} onChange={(event) => setSelectedTemplateId(event.target.value)}>
            {Object.entries(templateGroups).map(([group, templates]) => (
              <optgroup key={group} label={group}>
                {templates.map((template) => (
                  <option key={template.id} value={template.id}>
                    {template.name}
                  </option>
                ))}
              </optgroup>
            ))}
          </select>
          <button onClick={applyTemplate}>一键套模板</button>
        </div>
        <p className="muted">
          已选模板：{selectedTemplate.name} / {selectedTemplate.description}
        </p>
        <ul>
          {sites.map((site) => (
            <li key={site.id} className="site-item">
              <strong>{site.name}</strong>
              <span>{site.region}</span>
              <span>{site.status}</span>
              <span>ICP备案: {site.compliance?.icp_status ?? "not_started"}</span>
              <span>公安备案: {site.compliance?.psb_status ?? "not_started"}</span>
              <span>审核: {site.compliance?.review_status ?? "draft"}</span>
            </li>
          ))}
        </ul>
        <div className="site-item">
          <strong>模板配置面板</strong>
          <input
            value={templateConfig.siteName}
            onChange={(event) => setTemplateConfig((prev) => ({ ...prev, siteName: event.target.value }))}
            placeholder="站点名称"
          />
          <input
            value={templateConfig.headline}
            onChange={(event) => setTemplateConfig((prev) => ({ ...prev, headline: event.target.value }))}
            placeholder="首页标题"
          />
          <input
            value={templateConfig.subheadline}
            onChange={(event) => setTemplateConfig((prev) => ({ ...prev, subheadline: event.target.value }))}
            placeholder="首页说明"
          />
          <input
            value={templateConfig.ctaLabel}
            onChange={(event) => setTemplateConfig((prev) => ({ ...prev, ctaLabel: event.target.value }))}
            placeholder="按钮文案"
          />
          <button onClick={saveTemplateConfig}>保存模板配置</button>
        </div>
        <div className="site-item">
          <strong>备案中心</strong>
          <div className="action-row">
            <button onClick={() => submitCompliance("submit")}>提交审核</button>
            <button onClick={() => submitCompliance("approve")}>模拟通过</button>
            <button onClick={() => submitCompliance("reject")}>模拟驳回</button>
          </div>
          <select value={materialType} onChange={(event) => setMaterialType(event.target.value)}>
            {complianceMaterialTypes.map((item) => (
              <option key={item.value} value={item.value}>
                {item.label}
              </option>
            ))}
          </select>
          <input type="file" onChange={(event) => setUploadFile(event.target.files?.[0] ?? null)} />
          <button onClick={uploadComplianceMaterial}>上传材料</button>
          <input
            value={reviewNote}
            onChange={(event) => setReviewNote(event.target.value)}
            placeholder="审核意见 / 备注"
          />
          <div className="materials">
            {(activeSite?.compliance?.materials ?? []).map((material) => (
              <div key={material.id} className="material-card">
                <a href={material.public_url} target="_blank" rel="noreferrer">
                  {material.file_name}
                </a>
                <span>{material.type}</span>
                <span>{material.status}</span>
                <button onClick={() => reviewMaterial(material.id)}>标记已核验</button>
              </div>
            ))}
          </div>
          <div className="materials history">
            {(activeSite?.compliance?.review_history ?? []).map((event) => (
              <span key={event.id}>
                {event.action} / {event.actor} / {event.note || "-"}
              </span>
            ))}
          </div>
        </div>
      </section>
      <section className="list">
        <h2>页面与路线</h2>
        <ul>
          {pages.map((item) => (
            <li key={item.id}>
              {item.name}
              <span>{` /${item.slug}`}</span>
              <span>{item.status}</span>
              <button onClick={() => setHomepage(item.id)}>设为首页</button>
            </li>
          ))}
        </ul>
        <div className="roadmap">
          <h3>{roadmap?.product} 路线</h3>
          <p>{roadmap?.vision}</p>
          {(roadmap?.recommended ?? []).slice(0, 5).map((feature) => (
            <article key={feature.id} className="feature-card">
              <header>
                <strong>{feature.name}</strong>
                <span>{feature.phase}</span>
              </header>
              <p>{feature.summary}</p>
            </article>
          ))}
        </div>
      </section>
    </main>
  );
}

async function loadPages(siteId?: string): Promise<Page[]> {
  const query = siteId ? `?site_id=${encodeURIComponent(siteId)}` : "";
  const res = await fetch(`/api/pages${query}`);
  if (!res.ok) {
    return [];
  }
  return (await res.json()) as Page[];
}

async function loadSites(): Promise<Site[]> {
  const res = await fetch("/api/sites");
  if (!res.ok) {
    return [];
  }
  return (await res.json()) as Site[];
}

async function loadRoadmap(): Promise<Roadmap | null> {
  const res = await fetch("/api/platform/roadmap");
  if (!res.ok) {
    return null;
  }
  return (await res.json()) as Roadmap;
}

async function ensureAuth(): Promise<void> {
  await fetch("/api/login", {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username: "root", password: "root" }),
  });
}

async function authedPost(url: string): Promise<Response> {
  await ensureAuth();
  return await fetch(url, {
    method: "POST",
    credentials: "include",
  });
}

async function authedJsonFetch(url: string, body: unknown): Promise<Response> {
  await ensureAuth();
  return await fetch(url, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}
