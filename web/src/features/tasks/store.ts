import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { Task } from '@/api/tasks'
import { createTask, deleteTask, listTasks, setTaskCompleted } from '@/api/tasks'

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : '发生了未知错误'
}

export const useTaskStore = defineStore('tasks', () => {
  const items = ref<Task[]>([])
  const loading = ref(false)
  const saving = ref(false)
  const error = ref('')
  const completedCount = computed(() => items.value.filter(task => task.completed).length)

  async function load() {
    loading.value = true
    error.value = ''
    try {
      items.value = await listTasks()
    }
    catch (caught) {
      error.value = errorMessage(caught)
    }
    finally {
      loading.value = false
    }
  }

  async function add(title: string) {
    saving.value = true
    try {
      const task = await createTask(title)
      items.value.unshift(task)
      error.value = ''
      return task
    }
    catch (caught) {
      error.value = errorMessage(caught)
      throw caught
    }
    finally {
      saving.value = false
    }
  }

  async function toggle(task: Task, completed: boolean) {
    const updated = await setTaskCompleted(task.id, completed)
    const index = items.value.findIndex(item => item.id === task.id)
    if (index >= 0) {
      items.value[index] = updated
    }
  }

  async function remove(task: Task) {
    await deleteTask(task.id)
    items.value = items.value.filter(item => item.id !== task.id)
  }

  return { items, loading, saving, error, completedCount, load, add, toggle, remove }
})
