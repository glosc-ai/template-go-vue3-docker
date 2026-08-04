<script setup lang="ts">
import { ArrowRightIcon, BoxesIcon, ContainerIcon, DatabaseIcon, ServerCogIcon } from '@lucide/vue'
import AppHeader from '@/components/layout/AppHeader.vue'
import TaskBoard from '@/features/tasks/TaskBoard.vue'
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
</script>

<template>
  <div class="min-h-screen bg-background">
    <AppHeader />

    <main class="mx-auto flex max-w-6xl flex-col gap-16 px-4 py-14 sm:px-6 sm:py-20">
      <section class="flex max-w-3xl flex-col items-start gap-6">
        <Badge variant="outline">
          <BoxesIcon data-icon="inline-start" />
          Production-ready starter
        </Badge>
        <div class="flex flex-col gap-4">
          <h1 class="text-4xl font-semibold tracking-tight text-balance sm:text-6xl">
            从第一个业务功能开始，而不是从配置开始。
          </h1>
          <p class="max-w-2xl text-base leading-7 text-muted-foreground sm:text-lg">
            一个保持克制、可直接开发的 Go + Vue 3 + Docker 模板，内置数据库迁移、Redis、JWT、类型化 API 和持续集成。
          </p>
        </div>
        <div class="flex flex-wrap gap-3">
          <Button size="lg" as="a" href="#tasks">
            体验完整链路
            <ArrowRightIcon data-icon="inline-end" />
          </Button>
          <Button variant="outline" size="lg" as="a" href="#stack">
            查看目录设计
          </Button>
        </div>
      </section>

      <section id="stack" class="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader>
            <CardTitle>Go API</CardTitle>
            <CardDescription>标准库路由、结构化日志与优雅停机。</CardDescription>
            <CardAction><ServerCogIcon /></CardAction>
          </CardHeader>
          <CardContent>
            <p class="text-sm text-muted-foreground">
              领域包保持扁平，包含配置、健康检查、JWT 与可测试的任务 API。
            </p>
          </CardContent>
          <CardFooter><Badge variant="secondary">Go 1.25</Badge></CardFooter>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Vue 3 Web</CardTitle>
            <CardDescription>Vite、Pinia、Router 和 shadcn-vue。</CardDescription>
            <CardAction><ContainerIcon /></CardAction>
          </CardHeader>
          <CardContent>
            <p class="text-sm text-muted-foreground">
              API 类型集中管理，功能代码按领域放置，组件可按需继续扩展。
            </p>
          </CardContent>
          <CardFooter><Badge variant="secondary">Tailwind CSS v4</Badge></CardFooter>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Data & Docker</CardTitle>
            <CardDescription>默认 PostgreSQL，并兼容 MySQL 与 Redis。</CardDescription>
            <CardAction><DatabaseIcon /></CardAction>
          </CardHeader>
          <CardContent>
            <p class="text-sm text-muted-foreground">
              开发与生产镜像分离，迁移随 API 启动，数据卷默认持久化。
            </p>
          </CardContent>
          <CardFooter><Badge variant="secondary">Docker Compose</Badge></CardFooter>
        </Card>
      </section>

      <TaskBoard />
    </main>

    <Separator />
    <footer>
      <div class="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-8 text-sm text-muted-foreground sm:px-6">
        <span>Go Vue Starter</span>
        <span class="flex items-center gap-2"><DatabaseIcon /> Built for real projects</span>
      </div>
    </footer>
  </div>
</template>
