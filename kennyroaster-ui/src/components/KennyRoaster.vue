<template>
  <div class="page">
    <div class="main-container">
      <!-- Header -->
      <div class="header">
        <div class="logo" :style="appLogoStyle"></div>
        <h1 class="subtitle">
          "No need to win — just a pure evil thing. Bwak bwak bwaak!!"
        </h1>
      </div>

      <!-- Chat area -->
      <div class="chat-area" ref="chatArea">
        <div
          v-for="(m, idx) in messages"
          :key="idx"
          class="message"
          :class="m.role"
        >
          <div
            v-if="m.role === 'assistant'"
            class="avatar copilot"
            :style="assistantAvatarStyle"
          />
          <div class="message-content" v-html="m.html ?? escape(m.text)"></div>
          <div
            v-if="m.role === 'user'"
            class="avatar user"
            :style="userAvatarStyle"
          />
        </div>
      </div>

      <!-- Input -->
      <div class="input-area">
        <form class="input-container" @submit.prevent="send">
          <input
            v-model.trim="draft"
            type="text"
            placeholder="Message KennyRoaster..."
            @keydown.enter.exact.prevent="send"
          />
          <button type="submit" :disabled="!draft || sending">
            {{ sending ? 'Sending…' : 'Send' }}
          </button>
        </form>
      </div>

      <!-- Backdrop Spinner (overlay) using loader.png -->
      <div
        v-if="showBackdropLoader"
        class="loader-backdrop"
        role="alert"
        aria-live="polite"
        aria-busy="true"
      >
        <img class="loader-circle-img" :src="loaderPng" alt="Loading…" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, computed } from 'vue'
import kennyAvatar from '@/assets/kennyroaster.png'
import userAvatar from '@/assets/yaomeme.jpg'
import loaderPng from '@/assets/loader.png' // 🔥 use loader.png as the spinner

const appLogoStyle = computed(() => ({
  backgroundImage: `url(${kennyAvatar})`,
  backgroundSize: 'cover',
  backgroundPosition: 'center'
}))

const assistantAvatarStyle = computed(() => ({
  backgroundImage: `url(${kennyAvatar})`,
  backgroundSize: 'cover',
  backgroundPosition: 'center'
}))

const userAvatarStyle = computed(() => ({
  backgroundImage: `url(${userAvatar})`,
  backgroundSize: 'cover',
  backgroundPosition: 'center'
}))

type Role = 'assistant' | 'user'
interface ChatMessage {
  role: Role
  text?: string
  html?: string
}

const API_BASE = (import.meta.env.VITE_API_BASE as string | undefined)?.replace(/\/$/, '')

const messages = ref<ChatMessage[]>([
  { role: 'assistant', text: 'Who wants to get roasted?' },
])

const draft = ref('')
const sending = ref(false)
const chatArea = ref<HTMLDivElement | null>(null)

/** Backdrop spinner toggle */
const showBackdropLoader = ref(false)

/** Escape untrusted plain text */
function escape(s = ''): string {
  return s
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#039;')
}

/** Remove simple markdown emphasis */
function stripMarkdown(text: string): string {
  return text
    .replace(/\*\*(.*?)\*\*/g, '$1')
    .replace(/\*(.*?)\*/g, '$1')
    .replace(/__(.*?)__/g, '$1')
    .replace(/`([^`]+)`/g, '$1')
}

/**
 * Ensure Rank/Rate breaks
 */
function breakRankRate(text: string): string {
  let out = text
    .replace(/\s*Rank:/gi, '<br><br>Rank:')
    .replace(/\s*Rate:/gi, '<br><br>Rate:')
  out = out.replace(/(Rate:[^\n<]*)(?!<br>)/gi, '$1<br>')
  return out.trim()
}

/** Final formatter for assistant replies */
function formatAssistantToHtml(text: string): string {
  return breakRankRate(stripMarkdown(text))
}

async function send(): Promise<void> {
  if (!draft.value || sending.value) return

  const input = draft.value
  messages.value.push({ role: 'user', text: input })
  draft.value = ''

  await nextTick()
  scrollToBottom()

  sending.value = true
  showBackdropLoader.value = true // show overlay spinner

  try {
    const reply = await roastReply(input)
    messages.value.push({
      role: 'assistant',
      html: formatAssistantToHtml(reply),
    })
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : 'An unexpected error occurred.'
    messages.value.push({
      role: 'assistant',
      text: `⚠️ ${msg}`,
    })
  } finally {
    sending.value = false
    showBackdropLoader.value = false // hide overlay spinner
    await nextTick()
    scrollToBottom()
  }
}

type AskResponse = { answer?: string; [k: string]: unknown }

async function roastReply(input: string): Promise<string> {
  // If no backend configured, hit local /ask; otherwise adapt to `${API_BASE}/ask`
  if (!API_BASE) {
    const res = await fetch(`/ask`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ question: input }),
    })
    if (!res.ok) return `Backend error (${res.status})`
    const data = (await res.json()) as AskResponse
    return data.answer ?? 'No response from backend.'
  }

  // Local mock for quick testing
  if (/experiment/i.test(input)) {
    return `Experiment Day Hot Take: If ambition were caffeine, this is pure espresso—bold, jittery, slightly terrifying. Rank: 17 out of 48 Rate: 5/10`
  }
  return 'Consider yourself lightly toasted. 🔥'
}

function scrollToBottom(): void {
  if (chatArea.value) chatArea.value.scrollTop = chatArea.value.scrollHeight
}

onMounted(scrollToBottom)
</script>

<style scoped>
/* Page background */
.page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: linear-gradient(135deg, #0d1b2a 0%, #1a2f4a 100%);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
}

/* App shell */
.main-container {
  position: relative; /* required for absolute overlay */
  width: 100%;
  max-width: 900px;
  height: 1100px;
  display: flex;
  flex-direction: column;
  background: #0d1117;
  border-radius: 8px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  overflow: hidden;
  outline: 8px solid rgba(13, 27, 42, 0.15);
}

/* Header */
.header {
  text-align: center;
  padding: 20px 16px;
  border-bottom: 1px solid #30363d;
  background: #161b22;
}

.logo {
  height: 80px;
  width: 80px;
  border-radius: 50%;
  margin: 0 auto 8px;
  background-color: transparent;
}

/* Title */
.subtitle {
  color: #c9d1d9;
  font-size: 18px;
  font-weight: 300;
  margin: 0;
  font-family: Arial, sans-serif;
  font-style: italic;
}

/* Chat area */
.chat-area {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.message {
  display: flex;
  gap: 12px;
  animation: slideIn 0.3s ease;
  align-items: flex-start;
}

@keyframes slideIn {
  from { opacity: 0; transform: translateY(10px); }
  to   { opacity: 1; transform: translateY(0);   }
}

.message.user { justify-content: flex-end; }

/* Avatars */
.avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  flex-shrink: 0;
}

.avatar.copilot { background-color: transparent; }
.avatar.user    { background-color: transparent; }

/* Message bubbles */
.message-content {
  max-width: 60%;
  padding: 12px 16px;
  border-radius: 6px;
  line-height: 1.5;
  word-wrap: break-word;
  white-space: normal;
}

.message.assistant .message-content {
  background: #161b22;
  color: #c9d1d9;
}

.message.user .message-content {
  background: #238636;
  color: white;
}

/* Input area */
.input-area {
  border-top: 1px solid #30363d;
  padding: 16px 24px;
  background: #0d1117;
}

.input-container {
  display: flex;
  gap: 8px;
}

.input-container input {
  flex: 1;
  padding: 12px 16px;
  background: #0d1117;
  border: 1px solid #30363d;
  border-radius: 6px;
  color: #c9d1d9;
  font-size: 14px;
  transition: border-color 0.2s;
}

.input-container input:focus {
  outline: none;
  border-color: #1f6feb;
}

.input-container button {
  padding: 12px 16px;
  background: #1f6feb;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.2s;
}

.input-container button:hover {
  background: #388bfd;
}

.input-container button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* =========================
   Backdrop Spinner (overlay)
   ========================= */
.loader-backdrop {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;

  /* Dim + blur the app behind */
  background: rgba(0, 0, 0, 0.35);
  -webkit-backdrop-filter: blur(6px);
  backdrop-filter: blur(6px);

  z-index: 999;
}

/* Circular, spinning loader based on loader.png */
.loader-circle-img {
  width: 96px;
  height: 96px;
  border-radius: 50%;
  object-fit: cover;
  animation: spin 1s linear infinite;

  /* Slight glow/shadow for visibility */
  filter: drop-shadow(0 2px 8px rgba(0,0,0,0.4));
}

/* Respect users that prefer reduced motion */
@media (prefers-reduced-motion: reduce) {
  .loader-circle-img { animation: none; }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to   { transform: rotate(360deg); }
}
</style>