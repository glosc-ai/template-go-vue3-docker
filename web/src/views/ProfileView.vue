<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { LogOutIcon } from '@lucide/vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import { useAuthStore } from '@/features/auth/store'
import { confirmAction } from '@/lib/message'
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
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'

const auth = useAuthStore()
const router = useRouter()

function formatDateTime(value: string): string {
  if (!value) {
    return '—'
  }
  return new Intl.DateTimeFormat('zh-CN', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value))
}

async function signOut() {
  const alsoEndSSO = await confirmAction(
    '是否同时退出 Glosc AI 单点登录？选择取消仅退出本站。',
    '退出登录',
    { confirmText: '同时退出 SSO', cancelText: '仅退出本站' },
  )
  await auth.logout(alsoEndSSO)
  if (!alsoEndSSO) {
    router.replace('/login')
  }
}

onMounted(() => auth.ensureLoaded())
</script>

<template>
  <div class="min-h-screen bg-background">
    <AppHeader />

    <main class="mx-auto flex max-w-3xl flex-col gap-6 px-4 py-14 sm:px-6">
      <Card>
        <CardHeader>
          <CardTitle>账号信息</CardTitle>
          <CardDescription>
            这些字段来自 SSO 的 UserInfo 接口，并以 sub 为主键保存在本地 users 表。
          </CardDescription>
          <CardAction>
            <Badge variant="secondary">SSO 已绑定</Badge>
          </CardAction>
        </CardHeader>

        <CardContent class="flex flex-col gap-4">
          <div v-if="auth.loading && !auth.user" class="flex flex-col gap-3">
            <Skeleton v-for="index in 4" :key="index" class="h-10 w-full" />
          </div>

          <template v-else-if="auth.user">
            <div class="flex items-center gap-4">
              <img
                v-if="auth.user.avatar"
                :src="auth.user.avatar"
                :alt="`${auth.displayName} 的头像`"
                class="size-14 rounded-full border object-cover"
                referrerpolicy="no-referrer"
              >
              <span
                v-else
                class="grid size-14 place-items-center rounded-full bg-primary/10 text-lg font-medium text-primary"
                aria-hidden="true"
              >
                {{ auth.displayName.slice(0, 1) }}
              </span>
              <div class="flex flex-col">
                <span class="text-lg font-medium">{{ auth.displayName }}</span>
                <span class="text-sm text-muted-foreground">{{ auth.user.email || '未提供邮箱' }}</span>
              </div>
            </div>

            <Separator />

            <dl class="grid gap-4 sm:grid-cols-2">
              <div class="flex flex-col gap-1">
                <dt class="text-xs text-muted-foreground">
                  SSO 唯一标识（sub）
                </dt>
                <dd class="break-all font-mono text-sm">
                  {{ auth.user.subject }}
                </dd>
              </div>
              <div class="flex flex-col gap-1">
                <dt class="text-xs text-muted-foreground">
                  本地用户 ID
                </dt>
                <dd class="font-mono text-sm">
                  {{ auth.user.id }}
                </dd>
              </div>
              <div class="flex flex-col gap-1">
                <dt class="text-xs text-muted-foreground">
                  用户名
                </dt>
                <dd class="text-sm">
                  {{ auth.user.name || '—' }}
                </dd>
              </div>
              <div class="flex flex-col gap-1">
                <dt class="text-xs text-muted-foreground">
                  手机号
                </dt>
                <dd class="text-sm">
                  {{ auth.user.phone || '—' }}
                </dd>
              </div>
              <div class="flex flex-col gap-1">
                <dt class="text-xs text-muted-foreground">
                  首次登录
                </dt>
                <dd class="text-sm">
                  {{ formatDateTime(auth.user.created_at) }}
                </dd>
              </div>
              <div class="flex flex-col gap-1">
                <dt class="text-xs text-muted-foreground">
                  最近登录
                </dt>
                <dd class="text-sm">
                  {{ formatDateTime(auth.user.last_login_at) }}
                </dd>
              </div>
            </dl>
          </template>
        </CardContent>

        <CardFooter class="justify-between gap-4">
          <span class="text-xs text-muted-foreground">API：/api/v1/auth/session</span>
          <Button variant="outline" @click="signOut">
            <LogOutIcon data-icon="inline-start" />
            退出登录
          </Button>
        </CardFooter>
      </Card>
    </main>
  </div>
</template>
