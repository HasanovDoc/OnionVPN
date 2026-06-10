<template>
  <div class="flex flex-col items-center justify-center min-h-screen p-6 bg-slate-900 text-slate-100 relative">
    <h1 class="text-3xl font-bold mb-2 text-indigo-400">Onion VPN Utility</h1>
    <p class="text-sm text-slate-400 mb-6">Статус: 
      <span :class="isConnected ? 'text-green-400' : 'text-amber-400'">{{ status }}</span>
    </p>

    <div class="flex items-center gap-2 mb-4 bg-slate-800/50 px-4 py-2 rounded-xl border border-slate-700/60">
      <label class="relative inline-flex items-center cursor-pointer">
        <input 
          type="checkbox" 
          v-model="useSysProxy" 
          :disabled="isConnected" 
          @change="handleSysProxyChange"
          class="sr-only peer"
        >
        <div class="w-9 h-5 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-slate-100 after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-600"></div>
      </label>
      
      <span class="text-xs font-medium text-slate-300 flex items-center gap-1">
        Использовать<span class="text-indigo-400 font-bold">SysProxy</span> вместо модуля sing-box TUN

        <span class="relative group inline-flex items-center justify-center cursor-help ml-1 text-slate-400 hover:text-slate-200 transition-colors">
          <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" class="w-4 h-4">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9 5.25h.008v.008H12v-.008z" />
          </svg>

          <span class="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 hidden group-hover:block w-48 p-2 bg-slate-900 border border-slate-700 text-[11px] font-normal text-slate-300 rounded-lg shadow-xl z-10 text-center pointer-events-none">
            <span class="text-indigo-400 font-bold">SysProxy</span> автоматически настраивает параметры сети без создания виртуального сетевого адаптера. То есть discord, telegram и т.д не будт работать, какие-бы вы домены не указываели
            <span class="absolute top-full left-1/2 -translate-x-1/2 -mt-1 border-4 border-transparent border-t-slate-900"></span>
          </span>
        </span>
      </span>
    </div>
    <div class="flex items-center gap-2 mb-4 bg-slate-800/50 px-4 py-2 rounded-xl border border-slate-700/60">
      <label class="relative inline-flex items-center cursor-pointer">
        <input 
          type="checkbox" 
          v-model="isAutostartEnabled" 
          @change="handleAutostartChange"
          class="sr-only peer"
        >
        <div class="w-9 h-5 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-slate-100 after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-600"></div>
      </label>
      <span class="text-xs font-medium text-slate-300">Запускать OnionVPN при старте Windows</span>
    </div>

    <button 
      @click="toggleVpn"
      :class="isConnected ? 'bg-red-600 hover:bg-red-500' : 'bg-indigo-600 hover:bg-indigo-500'"
      class="px-6 py-3 rounded-lg font-medium transition-colors shadow-lg active:scale-95 mb-6"
    >
      {{ isConnected ? 'Отключить VPN' : 'Включить VPN' }}
    </button>

    <div class="w-full max-w-2xl mb-4 bg-slate-800 border border-slate-700 rounded-xl overflow-hidden">
      <div class="p-4 bg-slate-850">
        <label class="block text-sm font-medium text-indigo-400 mb-2">
          Сайты для туннелирования (выборочный роутинг):
        </label>
        
        <div class="mb-2 relative flex items-center bg-slate-900 border border-slate-700 rounded-lg pr-1 focus-within:border-indigo-500">
          <input 
            v-model="searchQuery"
            type="text"
            placeholder="Поиск, добавление или удаление доменов..."
            class="w-full bg-transparent px-3 py-1.5 text-xs text-slate-200 focus:outline-none placeholder-slate-500"
            @keydown.enter.prevent="addDomainFromSearch"
          />
          <div class="flex items-center gap-1.5 pr-1.5">
            <span v-if="searchQuery" @click="searchQuery = ''" class="text-slate-500 hover:text-slate-300 cursor-pointer text-xs px-1">✕</span>
            <button 
              v-if="searchQuery && isExactMatchMissing"
              @click="addDomainFromSearch"
              :disabled="isConnected"
              class="px-2 py-1 bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-[10px] font-medium text-white rounded transition-colors whitespace-nowrap"
              title="Добавить точный домен в список"
            >
              ➕ Добавить
            </button>
          </div>
        </div>

        <textarea
          v-if="!searchQuery"
          v-model="domainsText"
          :disabled="isConnected"
          placeholder="Пример:&#10;notion.so&#10;rutracker.org"
          class="w-full min-h-[5rem] h-40 bg-slate-950 border border-slate-700 rounded-lg p-2 font-mono text-xs text-slate-300 focus:outline-none focus:border-indigo-500 disabled:opacity-50 resize-y"
        ></textarea>

        <div v-else class="w-full h-40 bg-slate-950 border border-slate-700 rounded-lg p-2 font-mono text-xs overflow-y-auto block-inline">
          <div v-if="filteredDomains.length === 0" class="flex flex-col items-center justify-center h-full gap-1 py-4">
            <span class="text-slate-500 italic text-[11px]">Домен не найден</span>
          </div>

          <div v-else>
            <div v-for="(dom, i) in filteredDomains" :key="i" class="flex items-center justify-between py-1 px-1 border-b border-slate-900 last:border-0 hover:bg-slate-900/40 rounded">
              <div class="flex items-center gap-2">
                <span class="text-amber-400">{{ dom.name }}</span>
                <span :class="dom.enabled ? 'text-green-400 text-[10px]' : 'text-slate-500 text-[10px]'">
                  {{ dom.enabled ? '● активен' : '○ выключен' }}
                </span>
              </div>
              <button 
                @click="removeDomain(dom.name)"
                :disabled="isConnected"
                class="px-2 py-0.5 bg-red-950/60 hover:bg-red-600 disabled:opacity-50 text-red-400 hover:text-white border border-red-900/40 hover:border-transparent rounded text-[10px] transition-all"
                title="Удалить из списка"
              >
                ❌ Удалить
              </button>
            </div>
          </div>
        </div>

        <div class="flex justify-between items-center mt-2">
          <p class="text-[10px] text-slate-500">
            * Любые ресурсы на домене .onion будут направляться в Tor автоматически.
          </p>
          <button 
            @click="openModal"
            class="text-xs text-indigo-400 hover:text-indigo-300 transition-colors focus:outline-none"
          >
            ⚙ Расширенное
          </button>
        </div>
      </div>
    </div>

    <div class="w-full max-w-2xl mb-4 bg-slate-800 border border-slate-700 rounded-xl overflow-hidden">
      <button 
        @click="showBridgesConfig = !showBridgesConfig"
        class="w-full px-4 py-3 text-left font-medium flex justify-between items-center hover:bg-slate-750 transition-colors"
      >
        <span>Настройка мостов (Pluggable Transports)</span>
        <span class="text-xs text-indigo-400">{{ showBridgesConfig ? 'Скрыть' : 'Показать' }}</span>
      </button>
      
      <div v-if="showBridgesConfig" class="p-4 border-t border-slate-700 bg-slate-850">
        <label class="block text-xs text-slate-400 mb-2">
          Вставьте строки мостов (obfs4, snowflake или conjure), каждый с новой строки:
        </label>
        <textarea
          v-model="bridgesText"
          @input="handleBridgesInput"
          :disabled="isConnected"
          placeholder="obfs4 192.0.2.1:2443 ...&#10;snowflake 192.0.2.2:1234 ..."
          class="w-full h-24 bg-slate-950 border border-slate-700 rounded-lg p-2 font-mono text-xs text-slate-300 focus:outline-none focus:border-indigo-500 disabled:opacity-50 resize-y"
        ></textarea>
      </div>
    </div>

    <div class="w-full max-w-2xl bg-slate-950 rounded-xl border border-slate-800 relative group">
      <button 
        v-if="logs.length > 0"
        @click="clearLogs"
        class="absolute left-0 top-0 z-10 px-2 py-1 bg-slate-800/80 hover:bg-slate-700 text-[10px] font-medium rounded text-slate-400 hover:text-slate-200 transition-colors opacity-0 group-hover:opacity-100 focus:opacity-100"
        title="Очистить консоль"
      >
        Очистить
      </button>

      <div 
        ref="logConsole"
        @scroll="handleConsoleScroll"
        class="h-48 overflow-y-auto p-4 font-mono text-xs text-green-500 select-text selection:bg-indigo-500/40 selection:text-white"
      >
        <div v-for="(log, idx) in logs" :key="idx" class="whitespace-pre-wrap break-all leading-relaxed">
          {{ log }}
        </div>
      </div>
    </div>
    <button 
      @click="manualCheckUpdate"
      :disabled="isCheckingUpdate"
      class="absolute right-2 top-2 z-10 px-2.5 py-1 bg-slate-800/80 hover:bg-slate-700 disabled:opacity-50 text-[10px] font-medium rounded text-slate-400 hover:text-slate-200 border border-slate-700/50 transition-colors flex items-center gap-1"
      title="Проверить наличие обновлений"
    >
      <span v-if="isCheckingUpdate" class="w-2.5 h-2.5 border border-slate-400/30 border-t-slate-300 rounded-full animate-spin"></span>
      {{ isCheckingUpdate ? 'Проверка...' : 'Проверить обновления' }}
    </button>

    <div v-if="isModalOpen" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/70 backdrop-blur-sm">
      <div class="w-full max-w-lg bg-slate-800 border border-slate-700 rounded-xl shadow-2xl overflow-hidden flex flex-col max-h-[80vh]">
        <div class="p-4 bg-slate-850 border-b border-slate-700 flex flex-col gap-2">
          <div class="flex justify-between items-center">
            <h3 class="text-md font-medium text-indigo-400">Активность маршрутизации доменов</h3>
            <span class="text-xs text-slate-500">Показано: {{ filteredModalDomains.length }} из {{ tempStates.length }}</span>
          </div>
          
          <div class="relative mt-1">
            <input 
              v-model="modalSearchQuery"
              type="text"
              placeholder="Фильтр доменов в модалке..."
              class="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 placeholder-slate-500"
            />
            <span v-if="modalSearchQuery" @click="modalSearchQuery = ''" class="absolute right-2.5 top-1.5 text-slate-500 hover:text-slate-300 cursor-pointer text-xs">✕</span>
          </div>
        </div>

        <div class="p-4 bg-slate-900 flex-1 overflow-y-auto space-y-2">
          <div v-if="filteredModalDomains.length === 0" class="text-slate-500 text-center py-4 italic text-xs">
            Ничего не найдено
          </div>
          <div v-for="(dom, idx) in filteredModalDomains" :key="idx" class="flex items-center justify-between p-2 bg-slate-850 rounded-lg border border-slate-750">
            <span class="font-mono text-xs text-slate-200 truncate max-w-[70%]">{{ dom.name }}</span>
            <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" v-model="dom.enabled" class="sr-only peer" :disabled="isConnected">
              <div class="w-9 h-5 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-slate-100 after:border-slate-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-indigo-600"></div>
            </label>
          </div>
        </div>

        <div class="p-4 bg-slate-850 border-t border-slate-700 flex justify-end gap-3">
          <button 
            @click="closeModal" 
            class="px-4 py-2 bg-slate-700 hover:bg-slate-600 transition-colors text-xs font-medium rounded-lg"
          >
            Отмена
          </button>
          <button 
            @click="applyModalChanges" 
            class="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 transition-colors text-xs font-medium rounded-lg text-white"
          >
            Применить
          </button>
        </div>
      </div>
    </div>
    <div v-if="showUpdateModal" class="fixed inset-0 bg-black/70 flex items-center justify-center z-50 p-4">
      <div class="bg-slate-800 border border-slate-700 p-6 rounded-2xl max-w-md w-full shadow-2xl">
        <h3 class="text-xl font-bold text-indigo-400 mb-2">Доступно обновление!</h3>
        <p class="text-sm text-slate-300 mb-4">
          Доступна новая версия приложения: <span class="font-mono text-amber-400">{{ updateInfo.version }}</span>. Хотите обновить сейчас?
        </p>
        
        <div class="flex items-center gap-2 mb-6 select-none">
          <input 
            type="checkbox" 
            id="dontAskUpdate" 
            v-model="dontAskAgain"
            class="rounded bg-slate-700 border-slate-600 text-indigo-600 focus:ring-indigo-500"
          >
          <label for="dontAskUpdate" class="text-xs text-slate-400 cursor-pointer">
            Больше не спрашивать об обновлениях
          </label>
        </div>

        <div class="flex justify-end gap-3">
          <button 
            @click="closeUpdateModal" 
            class="px-4 py-2 text-sm font-medium text-slate-400 hover:text-slate-200 transition-colors"
            :disabled="isUpdating"
          >
            Нет
          </button>
          <button 
            @click="startUpdate" 
            class="px-5 py-2 text-sm font-medium bg-indigo-600 hover:bg-indigo-500 active:bg-indigo-700 text-white rounded-xl transition-all shadow-lg shadow-indigo-600/20 flex items-center gap-2"
            :disabled="isUpdating"
          >
            <span v-if="isUpdating" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
            {{ isUpdating ? 'Обновление...' : 'Да, обновить' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { ConnectToTor, DisconnectFromTor, CheckForUpdates, ApplyUpdate, LoadConfig, SaveConfig, ToggleAutostart, IsAutostartEnabled, MinimizeToTray } from '../wailsjs/go/main/App'

const isAutostartEnabled = ref(false)
const useSysProxy = ref(false)
const isConnected = ref(false)
const status = ref('Отключено')
const logs = ref([])
const showBridgesConfig = ref(false)
const isCheckingUpdate = ref(false)

const searchQuery = ref('')
const modalSearchQuery = ref('')

const isModalOpen = ref(false)
const tempStates = ref([])

const showUpdateModal = ref(false)
const dontAskAgain = ref(false)
const isUpdating = ref(false)
const updateInfo = ref({ version: '', downloadUrl: '' })

const bridgesText = ref('')
const domainStates = ref([])

const persistConfig = async () => {
  try {
    await SaveConfig(
      bridgesText.value, 
      JSON.parse(JSON.stringify(domainStates.value)), 
      useSysProxy.value
    )
  } catch (err) {
    console.error('Ошибка сохранения конфигурации:', err)
  }
}

const domainsText = computed({
  get() {
    return domainStates.value
      .map(d => d.name)
      .join('\n')
  },

  set(newText) {
    const lines = newText.split('\n')
    const currentDomains = []
    const seen = new Set()

    for (let line of lines) {
      let clean = line.trim().toLowerCase()
      clean = clean.replace(/^(https?:\/\/)?(www\.)?/, '')
      if (clean.includes('/')) {
        clean = clean.split('/')[0]
      }
      
      if (clean && !seen.has(clean)) {
        seen.add(clean)
        currentDomains.push(clean)
      }
    }

    const stateMap = new Map(domainStates.value.map(d => [d.name, d.enabled]))

    domainStates.value = currentDomains.map(name => ({
      name: name,
      enabled: stateMap.has(name) ? stateMap.get(name) : true
    }))

    persistConfig()
  }
})

const handleSysProxyChange = () => {
  persistConfig()
}

const logConsole = ref(null)
const userScrolledUp = ref(false)

const allDomainsMapped = computed(() => {
  return domainsText.value
    .split('\n')
    .map(d => d.trim())
    .filter(d => d.length > 0)
    .map(name => {
      const stateObj = domainStates.value.find(s => s.name === name)
      return {
        name,
        enabled: stateObj ? stateObj.enabled : true
      }
    })
})

const filteredDomains = computed(() => {
  if (!searchQuery.value.trim()) return []
  const query = searchQuery.value.toLowerCase().trim()
  return allDomainsMapped.value.filter(d => d.name.includes(query))
})

const isExactMatchMissing = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()
  if (!query) return false
  return !allDomainsMapped.value.some(d => d.name === query)
})

const filteredModalDomains = computed(() => {
  if (!modalSearchQuery.value.trim()) return tempStates.value
  const query = modalSearchQuery.value.toLowerCase().trim()
  return tempStates.value.filter(d => d.name.includes(query))
})

let connectionTimeout = null

onMounted(async () => {
  try {
    const config = await LoadConfig()
    if (config) {
      bridgesText.value = config.bridges || ''
      domainStates.value = config.domain_states || []
      useSysProxy.value = config.use_sys_proxy || false
    }
  } catch (err) {
    console.error('Не удалось загрузить конфигурацию из файла:', err)
  }

  try {
    isAutostartEnabled.value = await IsAutostartEnabled()
  } catch (err) {
    console.error(err)
  }

  EventsOn('tor:log', (line) => {
    logs.value.push(line)

    if (logs.value.length > 1000) {
      logs.value.shift()
    }
    scrollToBottom()
    
    if (line.includes('Bootstrapped 100%')) {
      if (connectionTimeout) clearTimeout(connectionTimeout)
      isConnected.value = true
      status.value = 'Защищено (Tor VPN Активен)'
    }

    const lowerLine = line.toLowerCase()
    if (
      lowerLine.includes('failed to bind') || 
      lowerLine.includes('could not launch') || 
      lowerLine.includes('connection refused') || 
      lowerLine.includes('bridge connection failed')
    ) {
      if (connectionTimeout) clearTimeout(connectionTimeout)
      if (!isConnected.value) {
        status.value = 'Ошибка: Мосты недоступны или заблокированы'
        DisconnectFromTor()
      }
    }
  })
  checkUpdateOnStartup()
})

const addDomainFromSearch = () => {
  const newDomain = searchQuery.value.trim().toLowerCase()
  if (!newDomain) return

  let clean = newDomain.replace(/"/g, '').replace(/,/g, '')
  clean = clean.replace(/^(https?:\/\/)?(www\.)?/, '')
  clean = clean.split('/')[0].split(':')[0]

  if (!clean) return

  const currentList = domainsText.value.split('\n').map(x => x.trim()).filter(Boolean)
  
  if (!currentList.includes(clean)) {
    currentList.push(clean)
    domainsText.value = currentList.join('\n')
    
    searchQuery.value = ''
  }
}

const removeDomain = (domainName) => {
  const currentList = domainsText.value.split('\n').map(x => x.trim()).filter(Boolean)
  const index = currentList.indexOf(domainName)
  
  if (index !== -1) {
    currentList.splice(index, 1)
    domainsText.value = currentList.join('\n')
    domainStates.value = domainStates.value.filter(s => s.name !== domainName)

    persistConfig()
  }
}

const openModal = () => {
  modalSearchQuery.value = ''
  tempStates.value = JSON.parse(JSON.stringify(allDomainsMapped.value))
  isModalOpen.value = true
}

const closeModal = () => {
  tempStates.value = []
  isModalOpen.value = false
}

const applyModalChanges = () => {
  domainStates.value = tempStates.value.map(t => ({ name: t.name, enabled: t.enabled }))
  closeModal()

  persistConfig()
}

const handleBridgesInput = () => {
  persistConfig()
}

const toggleVpn = async () => {
  if (!bridgesText.value) {
    status.value = 'Мостов не найдено, добавьте мосты'
    isConnected.value = false
    return
  }
  if (!isConnected.value) {
    status.value = 'Подключение...'
    searchQuery.value = ''
    
    const bridgesArray = bridgesText.value
      .split('\n')
      .map(line => line.trim())
      .filter(line => line.length > 0)

    // localStorage.setItem('vpn_use_sysproxy', useSysProxy.value.toString())

    const activeDomainsArray = allDomainsMapped.value
      .filter(d => d.enabled)
      .map(d => d.name)

    if (connectionTimeout) clearTimeout(connectionTimeout)
    
    if (bridgesArray.length > 0) {
      connectionTimeout = setTimeout(async () => {
        if (!isConnected.value) {
          status.value = 'Ошибка: Превышено время ожидания подключения (мосты не работают)'
          await DisconnectFromTor()
        }
      }, 45000)
    }

    try {
      const result = await ConnectToTor(bridgesArray, activeDomainsArray, useSysProxy.value)
    } catch (error) {
      if (connectionTimeout) clearTimeout(connectionTimeout)
      status.value = `Ошибка: ${result} Error: ${error}`
      isConnected.value = false
    }
  } else {
    if (connectionTimeout) clearTimeout(connectionTimeout)
    status.value = 'Отключение...'
    await DisconnectFromTor()
    isConnected.value = false
    status.value = 'Отключено'
  }
}

const scrollToBottom = () => {
  if (userScrolledUp.value || !logConsole.value) return

  setTimeout(() => {
    if (logConsole.value) {
      logConsole.value.scrollTop = logConsole.value.scrollHeight
    }
  }, 10)
}

const handleConsoleScroll = () => {
  if (!logConsole.value) return
  const el = logConsole.value
  const isAtBottom = el.scrollHeight - el.scrollTop <= el.clientHeight + 15
  userScrolledUp.value = !isAtBottom
}

const clearLogs = () => {
  logs.value = []
  userScrolledUp.value = false
}

const closeUpdateModal = () => {
  if (dontAskAgain.value) {
    localStorage.setItem('vpn_skip_updates', 'true')
  }
  showUpdateModal.value = false
}

const startUpdate = async () => {
  if (!updateInfo.value.downloadUrl) {
    alert('Ссылка на скачивание не найдена.')
    return
  }
  try {
    isUpdating.value = true
    await ApplyUpdate(updateInfo.value.downloadUrl)
  } catch (err) {
    alert(`Ошибка при обновлении: ${err}`)
    isUpdating.value = false
  }
}

const checkUpdateOnStartup = async () => {
  const isSkipped = localStorage.getItem('vpn_skip_updates') === 'true'
  if (isSkipped) return

  try {
    const res = await CheckForUpdates()
    if (res && res.hasUpdate) {
      updateInfo.value = {
        version: res.version,
        downloadUrl: res.downloadUrl
      }
      showUpdateModal.value = true
    }
  } catch (err) {
    console.error('Не удалось проверить обновления:', err)
  }
}

const handleAutostartChange = async () => {
  try {
    await ToggleAutostart(isAutostartEnabled.value)
  } catch (err) {
    console.error(err)
    isAutostartEnabled.value = !isAutostartEnabled.value
  }
}

const manualCheckUpdate = async () => {
  if (isCheckingUpdate.value) return
  
  try {
    isCheckingUpdate.value = true

    const res = await CheckForUpdates()
    
    if (res && res.hasUpdate) {
      updateInfo.value = {
        version: res.version,
        downloadUrl: res.downloadUrl
      }
      showUpdateModal.value = true
    } else {
      alert('У вас установлена самая актуальная версия приложения!')
    }
  } catch (err) {
    console.error('Ошибка при ручной проверке обновлений:', err)
    alert(`Не удалось проверить обновления: ${err}`)
  } finally {
    isCheckingUpdate.value = false
  }
}
</script>