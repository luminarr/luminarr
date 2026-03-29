import type { ReactNode } from "react";
import { ExternalLink } from "lucide-react";

interface PageHeaderProps {
  title: string;
  description: string;
  docsUrl?: string;
  action?: ReactNode;
}

export default function PageHeader({ title, description, docsUrl, action }: PageHeaderProps) {
  return (
    <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between", marginBottom: 24 }}>
      <div>
        <h1 style={{ margin: 0, fontSize: 20, fontWeight: 600, color: "var(--color-text-primary)", letterSpacing: "-0.01em" }}>
          {title}
        </h1>
        <p style={{ margin: "4px 0 0", fontSize: 13, color: "var(--color-text-secondary)" }}>
          {description}
          {docsUrl && (
            <>
              {" "}
              <a
                href={docsUrl}
                target="_blank"
                rel="noopener noreferrer"
                style={{
                  color: "var(--color-accent)",
                  textDecoration: "none",
                  fontSize: 13,
                  whiteSpace: "nowrap",
                }}
              >
                Learn more <ExternalLink size={12} style={{ verticalAlign: "-1px" }} />
              </a>
            </>
          )}
        </p>
      </div>
      {action}
    </div>
  );
}
