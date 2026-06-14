import { useEffect, useMemo, useState } from "react";

type Block = {
  id: string;
  type: "text" | "button" | "hero";
  content: string;
};

type Page = {
  id: string;
  name: string;
  slug: string;
  title: string;
  blocks: Block[];
};

const builtinBlocks: Block[] = [
  { id: "b1", type: "text", content: "主标题" },
  { id: "b2", type: "text", content: "正文说明" },
  { id: "b3", type: "button", content: "按钮" },
  { id: "b4", type: "hero", content: "Hero 区" },
];

export default function App() {
  const [pages, setPages] = useState<Page[]>([]);
  const [draft, setDraft] = useState<Block[]>([]);

  const previewHTML = useMemo(() => {
    if (draft.length === 0) {
      return "<p>拖拽左侧区块到右侧画布开始搭建页面</p>";
    }
    return draft
      .map((b) => {
        if (b.type === "button") {
          return `<a class='preview-btn'>${b.content}</a>`;
        }
        if (b.type === "hero") {
          return `<div class='preview-hero'>${b.content}</div>`;
        }
        return `<p>${b.content}</p>`;
      })
      .join("");
  }, [draft]);

  useEffect(() => {
    loadPages().then(setPages).catch(() => setPages([]));
  }, []);

  const onDrop = (block: Block) => {
    setDraft((prev) => [...prev, { ...block, id: `${block.id}-${Date.now()}` }]);
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
      name: "新建页面",
      slug: "",
      title: "新建页面",
      blocks: draft,
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
    await loadPages();
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
            {block.content}
          </button>
        ))}
      </section>
      <section className="canvas">
        <h2>画布</h2>
        <div className="frame" dangerouslySetInnerHTML={{ __html: previewHTML }} />
        <button onClick={saveDraft}>保存页面</button>
      </section>
      <section className="list">
        <h2>已建页面</h2>
        <ul>
          {pages.map((item) => (
            <li key={item.id}>
              {item.name}
              <span>{` /${item.slug}`}</span>
            </li>
          ))}
        </ul>
      </section>
    </main>
  );
}

async function loadPages(): Promise<Page[]> {
  const res = await fetch("/api/pages");
  if (!res.ok) {
    return [];
  }
  return (await res.json()) as Page[];
}
