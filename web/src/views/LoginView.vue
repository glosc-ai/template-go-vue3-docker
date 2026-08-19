<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { AlertCircleIcon, LogInIcon, ShieldCheckIcon } from '@lucide/vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import { useAuthStore } from '@/features/auth/store'
import { notifyError } from '@/lib/message'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

// The callback redirects here with ?error=<reason> when a login fails.
const failureReasons: Record<string, string> = {
  access_denied: '你取消了本次授权。',
  expired_state: '登录请求已过期，请重新发起登录。',
  invalid_response: '身份服务返回的信息不完整，请重试。',
  exchange_failed: '换取访问令牌失败，请确认客户端配置与回调地址。',
  userinfo_failed: '读取用户信息失败，请稍后重试。',
  internal_error: '服务端处理登录时出错，请稍后重试。',
}

const redirectTo = computed(() => {
  const target = route.query.redirect_to
  // Require a same-site root-relative path. `startsWith('/')` alone also
  // accepts `//evil.com/...`, a protocol-relative URL that browsers resolve
  // against a different origin — reject that case explicitly to match the
  // strictness of the backend's sso.safeRedirect.
  const isSameSitePath = typeof target === 'string' && target.startsWith('/') && !target.startsWith('//')
  return isSameSitePath ? target : '/profile'
})

const failure = computed(() => {
  const reason = route.query.error
  if (typeof reason !== 'string' || reason === '') {
    return ''
  }
  return failureReasons[reason] ?? `登录未完成（${reason}）。`
})

onMounted(async () => {
  if (failure.value) {
    notifyError(failure.value)
  }
  await auth.ensureLoaded()
  if (auth.isAuthenticated) {
    router.replace(redirectTo.value)
  }
})
</script>

<template>
  <div class="min-h-screen bg-background">
    <AppHeader />

    <main class="mx-auto flex max-w-md flex-col gap-6 px-4 py-16 sm:px-6">
      <Card>
        <CardHeader>
          <CardTitle>登录</CardTitle>
          <CardDescription>
            使用 Glosc AI 账号登录，授权完成后会返回本站。
          </CardDescription>
        </CardHeader>

        <CardContent class="flex flex-col gap-4">
          <Alert v-if="failure" variant="destructive">
            <AlertCircleIcon />
            <AlertTitle>登录未完成</AlertTitle>
            <AlertDescription>{{ failure }}</AlertDescription>
          </Alert>

          <Alert v-if="auth.unavailable">
            <AlertCircleIcon />
            <AlertTitle>服务端未配置单点登录</AlertTitle>
            <AlertDescription>
              请在服务端设置 SSO_CLIENT_ID、SSO_CLIENT_SECRET 与 SSO_REDIRECT_URL。
            </AlertDescription>
          </Alert>

          <p class="text-sm text-muted-foreground">
            登录过程采用 OAuth 2.0 授权码模式与 PKCE，客户端密钥仅保存在服务端。
          </p>

          <Button
            size="lg"
            class="w-full"
            :disabled="auth.loading || auth.unavailable"
            @click="auth.login(redirectTo)"
          >
            <Spinner v-if="auth.loading" data-icon="inline-start" />
            <LogInIcon v-else data-icon="inline-start" />
            使用 Glosc AI 账号登录
          </Button>
        </CardContent>

        <CardFooter class="text-xs text-muted-foreground">
          <span class="flex items-center gap-2">
            <ShieldCheckIcon class="size-4" />
            会话使用 HttpOnly Cookie 保存，前端无法读取令牌。
          </span>
        </CardFooter>
      </Card>
    </main>
  </div>
</template>
