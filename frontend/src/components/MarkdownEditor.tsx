import { useEffect, useRef, useState } from "react";
import { EditorContent, useEditor, type Editor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { Markdown } from "tiptap-markdown";

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
  const saveRef = useRef(onSave);
  const changeRef = useRef(onChange);
  saveRef.current = onSave;
  changeRef.current = onChange;

  const editor = useEditor({
    extensions: [StarterKit, Markdown.configure({ html: false })],
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
    <div className="md-editor">
      {empty && placeholder && <div className="md-placeholder">{placeholder}</div>}
      <EditorContent editor={editor} />
    </div>
  );
}
