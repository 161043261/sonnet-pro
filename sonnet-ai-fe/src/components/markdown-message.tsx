import React from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import { oneLight } from "react-syntax-highlighter/dist/esm/styles/prism";

interface MarkdownMessageProps {
  content: string;
}

function MarkdownMessage({ content }: MarkdownMessageProps) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        code({
          inline,
          className,
          children,
          ...props
        }: React.HTMLAttributes<HTMLElement> & { inline?: boolean }) {
          const match = /language-(\w+)/.exec(className || "");
          return !inline && match ? (
            <SyntaxHighlighter
              {...props}
              style={oneLight}
              language={match[1]}
              PreTag="div"
              className="my-2 rounded-md"
            >
              {String(children).replace(/\n$/, "")}
            </SyntaxHighlighter>
          ) : (
            <code
              {...props}
              className={`${className} bg-base-200 rounded px-1 py-0.5 text-sm`}
            >
              {children}
            </code>
          );
        },
        p({ children }) {
          return <p className="mb-2 leading-relaxed last:mb-0">{children}</p>;
        },
        ul({ children }) {
          return <ul className="mb-2 list-disc pl-5">{children}</ul>;
        },
        ol({ children }) {
          return <ol className="mb-2 list-decimal pl-5">{children}</ol>;
        },
        li({ children }) {
          return <li className="mb-1">{children}</li>;
        },
        a({ href, children }) {
          return (
            <a
              href={href}
              target="_blank"
              rel="noreferrer"
              className="link link-primary"
            >
              {children}
            </a>
          );
        },
        h1({ children }) {
          return <h1 className="mt-4 mb-3 text-2xl font-bold">{children}</h1>;
        },
        h2({ children }) {
          return <h2 className="mt-3 mb-2 text-xl font-bold">{children}</h2>;
        },
        h3({ children }) {
          return <h3 className="mt-2 mb-2 text-lg font-bold">{children}</h3>;
        },
        table({ children }) {
          return (
            <div className="my-4 overflow-x-auto">
              <table className="table-zebra table-sm table">{children}</table>
            </div>
          );
        },
      }}
    >
      {content}
    </ReactMarkdown>
  );
}

export default MarkdownMessage;
