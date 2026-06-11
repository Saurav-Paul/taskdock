import { useEffect, useRef, useState } from "react";
import { EditorContent, useEditor, type Editor } from "@tiptap/react";
import Image from "@tiptap/extension-image";
import StarterKit from "@tiptap/starter-kit";
import Table from "@tiptap/extension-table";
import TableRow from "@tiptap/extension-table-row";
import TableCell from "@tiptap/extension-table-cell";
import TableHeader from "@tiptap/extension-table-header";
import { Markdown } from "tiptap-markdown";
import { uploadAttachment } from "../api";
import { MermaidCodeBlock } from "./MermaidCodeBlock";

function getMarkdown(editor: Editor): string {
  return (editor.storage as Record<string, any>).markdown.getMarkdown();
}

interface Props {
  value: string;
  placeholder?: string;
  autofocus?: boolean;
  /** Called on blur and on cmd+enter with the markdown source. */
  onSave?: (markdown: string) => void;
  /** Called on every change with the markdown source. */
  onChange?: (markdown: string) => void;
}

export function MarkdownEditor({ value, placeholder, autofocus, onSave, onChange }: Props) {
  const [empty, setEmpty] = useState(value.trim() === "");
  const [uploading, setUploading] = useState(false);
  const saveRef = useRef(onSave);
  const changeRef = useRef(onChange);
  saveRef.current = onSave;
  changeRef.current = onChange;

  async function uploadImages(files: File[], pos?: number) {
    const images = files.filter((f) => f.type.startsWith("image/"));
    if (images.length < files.length) {
      alert("Only image files (png, jpeg, gif, webp, svg) can be attached.");
    }
    if (images.length === 0) return;
    setUploading(true);
    try {
      for (const file of images) {
        const { url } = await uploadAttachment(file);
        const node = { type: "image", attrs: { src: url } };
        if (pos != null) editor?.chain().insertContentAt(pos, node).run();
        else editor?.chain().focus().insertContent(node).run();
      }
    } catch (e) {
      alert(`Image upload failed: ${(e as Error).message}`);
    } finally {
      setUploading(false);
    }
  }

  const editor = useEditor({
    extensions: [
      StarterKit.configure({ codeBlock: false }),
      MermaidCodeBlock,
      Image.configure({ inline: false }),
      Table.configure({ resizable: false }),
      TableRow,
      TableCell,
      TableHeader,
      Markdown.configure({ html: false }),
    ],
    content: value,
    autofocus: autofocus ? "end" : false,
    onUpdate: ({ editor }) => {
      setEmpty(editor.isEmpty);
      changeRef.current?.(getMarkdown(editor));
    },
    onBlur: ({ editor }) => {
      saveRef.current?.(getMarkdown(editor));
    },
    editorProps: {
      handleKeyDown: (_view, event) => {
        if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
          if (saveRef.current) {
            saveRef.current(getMarkdown(editor!));
            editor?.commands.blur();
          }
          return true;
        }
        if (event.key === "Escape") {
          editor?.commands.blur();
          return true;
        }
        return false;
      },
      handlePaste: (_view, event) => {
        const files = Array.from(event.clipboardData?.files ?? []);
        if (files.length === 0) return false;
        event.preventDefault();
        void uploadImages(files);
        return true;
      },
      handleDrop: (view, event) => {
        const files = Array.from(event.dataTransfer?.files ?? []);
        if (files.length === 0) return false;
        event.preventDefault();
        const coords = view.posAtCoords({ left: event.clientX, top: event.clientY });
        void uploadImages(files, coords?.pos);
        return true;
      },
    },
  });

  // Sync external value changes (e.g. switching issues) into the editor.
  useEffect(() => {
    if (!editor) return;
    if (getMarkdown(editor) !== value && !editor.isFocused) {
      editor.commands.setContent(value);
      setEmpty(editor.isEmpty);
    }
  }, [value, editor]);

  return (
    <div className={`md-editor ${uploading ? "uploading" : ""}`}>
      {empty && placeholder && <div className="md-placeholder">{placeholder}</div>}
      {uploading && <div className="md-uploading">Uploading image…</div>}
      <EditorContent editor={editor} />
    </div>
  );
}
