'use client'

import {useState} from 'react'
import {Pencil, Loader2} from 'lucide-react'
import {toast} from 'sonner'
import {Button} from '@/components/ui/button'
import {Textarea} from '@/components/ui/textarea'
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from '@/components/ui/dialog'

type Props<T> = { title: string; description: string; value: T; onSave: (value: unknown) => Promise<boolean> }

export function RawJsonEditor<T>({title, description, value, onSave}: Props<T>) {
  const [open, setOpen] = useState(false)
  const [text, setText] = useState('')
  const [saving, setSaving] = useState(false)
  const save = async () => {
    let parsed: unknown
    try { parsed = JSON.parse(text) } catch { toast.error('JSON 格式错误，请检查配置'); return }
    setSaving(true)
    try { if (await onSave(parsed)) setOpen(false) } finally { setSaving(false) }
  }
  return <>
    <Button type="button" variant="ghost" size="icon-xs" aria-label={`编辑 ${title}`} onClick={() => { setText(JSON.stringify(value, null, 2)); setOpen(true) }}><Pencil/></Button>
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-[650px]">
        <DialogHeader><DialogTitle>编辑 {title} JSON</DialogTitle><DialogDescription>{description}</DialogDescription></DialogHeader>
        <Textarea value={text} onChange={(event) => setText(event.target.value)} className="min-h-[320px] font-mono text-xs" disabled={saving} spellCheck={false}/>
        <DialogFooter><Button type="button" variant="outline" onClick={() => setOpen(false)} disabled={saving}>取消</Button><Button type="button" onClick={save} disabled={saving}>{saving && <Loader2 className="animate-spin"/>}保存</Button></DialogFooter>
      </DialogContent>
    </Dialog>
  </>
}
