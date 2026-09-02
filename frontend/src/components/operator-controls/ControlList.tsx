import { Trash2 } from "lucide-react";
import type { ReactNode } from "react";
import { Empty } from "../Status";

export function ControlList<T>({ items, render, empty }: { items: T[]; render: (item: T) => ReactNode; empty: string }) {
  if (!items.length) return <Empty title={empty} detail="Operator controls appear here after they are created." />;
  return <div className="control-list">{items.map((item, index) => <article key={index}>{render(item)}</article>)}</div>;
}

export function DeleteControl({ action }: { action: () => void }) {
  return <button className="icon-button danger" aria-label="Delete operator control" onClick={action}><Trash2 /></button>;
}
