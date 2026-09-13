# Graph Report - ui  (2026-09-13)

## Corpus Check
- 76 files · ~748,493 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 478 nodes · 1085 edges · 17 communities (14 shown, 3 thin omitted)
- Extraction: 100% EXTRACTED · 0% INFERRED · 0% AMBIGUOUS · INFERRED: 1 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `d22e0b84`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]

## God Nodes (most connected - your core abstractions)
1. `cn()` - 117 edges
2. `Button()` - 21 edges
3. `compilerOptions` - 16 edges
4. `ToolBadge()` - 9 edges
5. `request()` - 9 edges
6. `getArg()` - 8 edges
7. `getToolKind()` - 8 edges
8. `ToolEvent` - 8 edges
9. `getToolContent()` - 7 edges
10. `Item()` - 7 edges

## Surprising Connections (you probably didn't know these)
- `cn()` --calls--> `clsx`  [INFERRED]
  src/lib/utils.ts → package.json
- `FileCard()` --calls--> `cn()`  [EXTRACTED]
  src/components/attachments-message.tsx → src/lib/utils.ts
- `ToolRow()` --calls--> `cn()`  [EXTRACTED]
  src/components/chat-message.tsx → src/lib/utils.ts
- `StepBlock()` --calls--> `cn()`  [EXTRACTED]
  src/components/chat-message.tsx → src/lib/utils.ts
- `ItemSeparator()` --calls--> `cn()`  [EXTRACTED]
  src/components/ui/item.tsx → src/lib/utils.ts

## Communities (17 total, 3 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.06
Nodes (60): useIsMobile(), cn(), WEEK_DAYS, Avatar(), AvatarBadge(), AvatarFallback(), AvatarGroup(), AvatarGroupCount() (+52 more)

### Community 1 - "Community 1"
Cohesion: 0.07
Nodes (63): configApi, API_CONFIG, ApiError, createSSEStream(), del(), fetchWithTimeout(), get(), handleErrorResponse() (+55 more)

### Community 2 - "Community 2"
Cohesion: 0.06
Nodes (44): PlanStep, ToolEvent, AttachmentsMessage(), AttachmentsMessageProps, FileCard(), ChatMessage(), ChatMessageProps, StepBlock() (+36 more)

### Community 3 - "Community 3"
Cohesion: 0.09
Nodes (36): ListA2AServerItem, ListMCPServerItem, MCPServerConfig, DeleteSessionDialogProps, GlobalHeader(), A2ASettingProps, CommonSettingProps, ManusSettings() (+28 more)

### Community 4 - "Community 4"
Cohesion: 0.06
Nodes (35): sessionApi, Session, DeleteSessionDialog(), RenameSessionDialog(), SessionItem(), SessionItemProps, SessionList(), formatRelativeDate() (+27 more)

### Community 5 - "Community 5"
Cohesion: 0.05
Nodes (41): dependencies, class-variance-authority, clsx, lucide-react, next, next-themes, @novnc/novnc, @radix-ui/react-avatar (+33 more)

### Community 6 - "Community 6"
Cohesion: 0.12
Nodes (20): A2aTool(), A2aToolProps, BashTool(), BashToolProps, BrowserTool(), BrowserToolProps, DefaultTool(), DefaultToolProps (+12 more)

### Community 7 - "Community 7"
Cohesion: 0.15
Nodes (14): fileApi, FileInfo, ChatInput, ChatInputProps, ChatInputRef, FilePreviewPanel(), FilePreviewPanelProps, isSupportedFileType() (+6 more)

### Community 8 - "Community 8"
Cohesion: 0.10
Nodes (19): compilerOptions, allowJs, esModuleInterop, incremental, isolatedModules, jsx, lib, module (+11 more)

### Community 9 - "Community 9"
Cohesion: 0.11
Nodes (18): aliases, components, hooks, lib, ui, utils, iconLibrary, registries (+10 more)

### Community 10 - "Community 10"
Cohesion: 0.14
Nodes (13): API 调用, code:block1 (ui/), code:bash (# 安装依赖), code:bash (npm run build), Docker 部署, Manus 前端 UI, 安装与启动, 技术栈 (+5 more)

### Community 11 - "Community 11"
Cohesion: 0.22
Nodes (7): metadata, LeftPanel(), ModelsContext, ModelsContextValue, ModelsProvider(), SessionsProvider(), Toaster()

### Community 12 - "Community 12"
Cohesion: 0.25
Nodes (5): VNCOverlay(), VNCOverlayProps, VNCViewer, VNCStatus, VNCViewerProps

## Knowledge Gaps
- **141 isolated node(s):** `config`, `name`, `version`, `private`, `dev` (+136 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **3 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `cn()` connect `Community 0` to `Community 2`, `Community 3`, `Community 4`, `Community 5`, `Community 7`?**
  _High betweenness centrality (0.337) - this node is a cross-community bridge._
- **Why does `clsx` connect `Community 5` to `Community 0`?**
  _High betweenness centrality (0.135) - this node is a cross-community bridge._
- **What connects `config`, `name`, `version` to the rest of the system?**
  _141 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.05844155844155844 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.06773211567732115 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.06077694235588972 - nodes in this community are weakly interconnected._
- **Should `Community 3` be split into smaller, more focused modules?**
  _Cohesion score 0.08705882352941176 - nodes in this community are weakly interconnected._