<script setup lang="ts">
import { Code2Icon, Layers3Icon } from '@lucide/vue'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
</script>

<template>
  <header class="sticky top-0 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
    <div class="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 sm:px-6">
      <a class="flex items-center gap-2 font-medium" href="/">
        <span class="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <Layers3Icon />
        </span>
        <span>Go Vue Starter</span>
      </a>

      <nav class="flex items-center gap-1" aria-label="主导航">
        <Button variant="ghost" size="sm" as="a" href="#stack">
          技术栈
        </Button>
        <Button variant="ghost" size="sm" as="a" href="#tasks">
          示例
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
