<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { Code2Icon, Layers3Icon, LogInIcon, UserRoundIcon } from '@lucide/vue'
import { useAuthStore } from '@/features/auth/store'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { Spinner } from '@/components/ui/spinner'

const auth = useAuthStore()

onMounted(() => auth.ensureLoaded())
</script>

<template>
  <header class="sticky top-0 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
    <div class="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 sm:px-6">
      <RouterLink class="flex items-center gap-2 font-medium" to="/">
        <span class="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <Layers3Icon />
        </span>
        <span>Go Vue Starter</span>
      </RouterLink>

      <nav class="flex items-center gap-1" aria-label="主导航">
        <Button variant="ghost" size="sm" as="a" href="/#stack">
          技术栈
        </Button>
        <Button variant="ghost" size="sm" as="a" href="/#tasks">
          示例
        </Button>

        <Spinner v-if="auth.loading && !auth.resolved" class="mx-2" aria-label="正在确认登录状态" />

        <Button
          v-else-if="auth.isAuthenticated"
          variant="ghost"
          size="sm"
          as-child
        >
          <RouterLink to="/profile">
            <UserRoundIcon data-icon="inline-start" />
            {{ auth.displayName }}
          </RouterLink>
        </Button>

        <Button v-else variant="outline" size="sm" as-child>
          <RouterLink to="/login">
            <LogInIcon data-icon="inline-start" />
            登录
          </RouterLink>
        </Button>

        <Button
          variant="ghost"
          size="icon"
          as="a"
          href="https://github.com/gloscai/template-go-vue3-docker"
          target="_blank"
          rel="noreferrer"
          aria-label="查看 GitHub 仓库"
        >
          <Code2Icon />
        </Button>
      </nav>
    </div>
    <Separator />
  </header>
</template>
