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

const builtinBlocks: Block[] = [
  { id: "b1", type: "text", label: "主标题", props: { text: "主标题" } },
  { id: "b2", type: "text", label: "正文说明", props: { text: "正文说明" } },
  { id: "b3", type: "button", label: "按钮", props: { label: "立即咨询" } },
  { id: "b4", type: "hero", label: "Hero 区", props: { headline: "这是 Hero 区" } },
];

export default function App() {
  const [pages, setPages] = useState<Page[]>([]);
  const [draft, setDraft] = useState<Block[]>([]);
  const [sites, setSites] = useState<Site[]>([]);
  const [roadmap, setRoadmap] = useState<Roadmap | null>(null);
  const [activeSiteId, setActiveSiteId] = useState("site-default");
  const [selectedTemplateId, setSelectedTemplateId] = useState("tpl-hero");
  const [uploadFile, setUploadFile] = useState<File | null>(null);
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
    form.append("type", "business-license");
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
            <option value="tpl-hero">企业官网模板</option>
            <option value="tpl-product">产品展示模板</option>
          </select>
          <button onClick={applyTemplate}>一键套模板</button>
        </div>
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
          </div>
          <input type="file" onChange={(event) => setUploadFile(event.target.files?.[0] ?? null)} />
          <button onClick={uploadComplianceMaterial}>上传材料</button>
          <div className="materials">
            {(activeSite?.compliance?.materials ?? []).map((material) => (
              <a key={material.id} href={material.public_url} target="_blank" rel="noreferrer">
                {material.file_name} / {material.status}
              </a>
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
