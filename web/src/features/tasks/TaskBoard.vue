<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { AlertCircleIcon, ListChecksIcon, PlusIcon, Trash2Icon } from '@lucide/vue'
import { toast } from 'vue-sonner'
import type { Task } from '@/api/tasks'
import { useTaskStore } from './store'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Field, FieldError, FieldGroup, FieldLabel, FieldLegend, FieldSet } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { Spinner } from '@/components/ui/spinner'
import { cn } from '@/lib/utils'

const store = useTaskStore()
const title = ref('')
const validationError = computed(() => title.value.length > 160 ? '任务标题不能超过 160 个字符。' : '')

async function submit() {
  const trimmed = title.value.trim()
  if (!trimmed) {
    toast.error('请输入任务标题')
    return
  }
  if (validationError.value) {
    return
  }
  try {
    await store.add(trimmed)
    title.value = ''
    toast.success('任务已创建')
  }
  catch {
    toast.error('创建失败，请确认 API 与数据库已启动')
  }
}

async function toggle(task: Task, completed: boolean) {
  try {
    await store.toggle(task, completed)
  }
  catch {
    toast.error('更新任务失败')
  }
}

async function remove(task: Task) {
  try {
    await store.remove(task)
    toast.success('任务已删除')
  }
  catch {
    toast.error('删除任务失败')
  }
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

onMounted(store.load)
</script>

<template>
  <Card id="tasks">
    <CardHeader>
      <CardTitle>端到端任务示例</CardTitle>
      <CardDescription>
        每一次操作都会经过 Vue、Go API，并最终写入数据库。
      </CardDescription>
      <CardAction>
        <Badge variant="secondary">
          {{ store.completedCount }} / {{ store.items.length }} 已完成
        </Badge>
      </CardAction>
    </CardHeader>

    <CardContent class="flex flex-col gap-5">
      <form @submit.prevent="submit">
        <FieldGroup>
          <Field :data-invalid="Boolean(validationError)">
            <FieldLabel for="task-title">
              新任务
            </FieldLabel>
            <div class="flex flex-col gap-2 sm:flex-row">
              <Input
                id="task-title"
                v-model="title"
                :aria-invalid="Boolean(validationError)"
                maxlength="161"
                placeholder="例如：添加第一个业务模块"
                autocomplete="off"
              />
              <Button type="submit" :disabled="store.saving || Boolean(validationError)">
                <Spinner v-if="store.saving" data-icon="inline-start" />
                <PlusIcon v-else data-icon="inline-start" />
                添加任务
              </Button>
            </div>
            <FieldError v-if="validationError">
              {{ validationError }}
            </FieldError>
          </Field>
        </FieldGroup>
      </form>

      <Separator />

      <Alert v-if="store.error" variant="destructive">
        <AlertCircleIcon />
        <AlertTitle>暂时无法连接后端</AlertTitle>
        <AlertDescription>
          {{ store.error }}。请运行 make dev，或检查 VITE_API_PROXY_TARGET。
        </AlertDescription>
      </Alert>

      <div v-if="store.loading" class="flex flex-col gap-3" aria-label="正在加载任务">
        <Skeleton v-for="index in 3" :key="index" class="h-12 w-full" />
      </div>

      <Empty v-else-if="store.items.length === 0">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <ListChecksIcon />
          </EmptyMedia>
          <EmptyTitle>还没有任务</EmptyTitle>
          <EmptyDescription>
            添加第一条任务，即可验证前端、API 与数据库的完整链路。
          </EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button variant="outline" type="button" @click="title = '完成项目初始化'">
            使用示例标题
          </Button>
        </EmptyContent>
      </Empty>

      <FieldSet v-else>
        <FieldLegend class="sr-only">
          任务列表
        </FieldLegend>
        <FieldGroup>
          <Field
            v-for="task in store.items"
            :key="task.id"
            orientation="horizontal"
          >
            <Checkbox
              :id="`task-${task.id}`"
              :model-value="task.completed"
              :aria-label="`标记 ${task.title} 为${task.completed ? '未完成' : '已完成'}`"
              @update:model-value="value => toggle(task, value === true)"
            />
            <FieldLabel :for="`task-${task.id}`" class="min-w-0 flex-1">
              <span :class="cn('truncate', task.completed && 'text-muted-foreground line-through')">
                {{ task.title }}
              </span>
              <span class="shrink-0 text-xs text-muted-foreground">
                {{ formatDate(task.created_at) }}
              </span>
            </FieldLabel>
            <Button
              variant="ghost"
              size="icon-sm"
              type="button"
              :aria-label="`删除 ${task.title}`"
              @click="remove(task)"
            >
              <Trash2Icon />
            </Button>
          </Field>
        </FieldGroup>
      </FieldSet>
    </CardContent>

    <CardFooter class="justify-between gap-4">
      <span class="text-xs text-muted-foreground">API：/api/v1/tasks</span>
      <span class="text-xs text-muted-foreground">PostgreSQL / MySQL</span>
    </CardFooter>
  </Card>
</template>
