import { request } from './client'

export interface Task {
  id: number
  title: string
  completed: boolean
  created_at: string
}

interface DataResponse<T> {
  data: T
}

export async function listTasks(): Promise<Task[]> {
  const response = await request<DataResponse<Task[]>>('/tasks')
  return response.data
}

export async function createTask(title: string): Promise<Task> {
  const response = await request<DataResponse<Task>>('/tasks', {
    method: 'POST',
    body: JSON.stringify({ title }),
  })
  return response.data
}

export async function setTaskCompleted(id: number, completed: boolean): Promise<Task> {
  const response = await request<DataResponse<Task>>(`/tasks/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ completed }),
  })
  return response.data
}

export async function deleteTask(id: number): Promise<void> {
  await request<void>(`/tasks/${id}`, { method: 'DELETE' })
}
