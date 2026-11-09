<script setup lang="ts">
import * as echarts from 'echarts'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { NL2SQLChartSpec } from '@/utils/nl2sql'

const props = defineProps<{
  spec: NL2SQLChartSpec
}>()

const el = ref<HTMLElement | null>(null)
let chart: echarts.ECharts | null = null

function buildOption(spec: NL2SQLChartSpec): echarts.EChartsOption | null {
  const rows = spec.rows ?? []
  if (!rows.length || spec.chartType === 'table') return null

  const xField = spec.xField
  const yField = spec.yFields?.[0]
  if (!xField || !yField) return null

  if (spec.chartType === 'line') {
    const seriesField = spec.seriesField
    if (seriesField && rows.some((r) => r[seriesField])) {
      const groups = new Map<string, Record<string, unknown>[]>()
      for (const r of rows) {
        const key = String(r[seriesField] ?? 'default')
        if (!groups.has(key)) groups.set(key, [])
        groups.get(key)!.push(r)
      }
      const xSet = new Set<string>()
      for (const r of rows) xSet.add(String(r[xField]))
      const xData = [...xSet].sort()
      const series = [...groups.entries()].map(([name, items]) => ({
        name,
        type: 'line' as const,
        smooth: true,
        data: xData.map((x) => {
          const hit = items.find((it) => String(it[xField]) === x)
          return hit ? Number(hit[yField]) : null
        }),
      }))
      return {
        title: { text: spec.title, left: 'center', textStyle: { fontSize: 14 } },
        tooltip: { trigger: 'axis' },
        legend: { bottom: 0 },
        grid: { left: 48, right: 24, top: 40, bottom: 48 },
        xAxis: { type: 'category', data: xData },
        yAxis: { type: 'value', scale: true },
        series,
      }
    }
    return {
      title: { text: spec.title, left: 'center', textStyle: { fontSize: 14 } },
      tooltip: { trigger: 'axis' },
      grid: { left: 48, right: 24, top: 40, bottom: 32 },
      xAxis: { type: 'category', data: rows.map((r) => String(r[xField])) },
      yAxis: { type: 'value', scale: true },
      series: [
        {
          type: 'line',
          smooth: true,
          data: rows.map((r) => Number(r[yField])),
          areaStyle: { opacity: 0.08 },
        },
      ],
    }
  }

  if (spec.chartType === 'bar') {
    return {
      title: { text: spec.title, left: 'center', textStyle: { fontSize: 14 } },
      tooltip: { trigger: 'axis' },
      grid: { left: 48, right: 24, top: 40, bottom: 32 },
      xAxis: { type: 'category', data: rows.map((r) => String(r[xField])) },
      yAxis: { type: 'value' },
      series: [{ type: 'bar', data: rows.map((r) => Number(r[yField])) }],
    }
  }
  return null
}

function render() {
  if (!el.value) return
  const opt = buildOption(props.spec)
  if (!opt) {
    chart?.dispose()
    chart = null
    return
  }
  if (!chart) chart = echarts.init(el.value)
  chart.setOption(opt, true)
  chart.resize()
}

onMounted(() => {
  render()
  window.addEventListener('resize', render)
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', render)
  chart?.dispose()
  chart = null
})

watch(() => props.spec, render, { deep: true })
</script>

<template>
  <div v-if="spec.chartType !== 'table' && (spec.rows?.length ?? 0) > 0" class="chart-wrap">
    <div ref="el" class="chart" />
  </div>
  <div v-if="(spec.rows?.length ?? 0) > 0" class="table-wrap">
    <div class="table-title">{{ spec.title || '查询结果' }}</div>
    <div class="table-scroll">
      <table>
        <thead>
          <tr>
            <th v-for="col in spec.columns ?? Object.keys(spec.rows![0] ?? {})" :key="col">
              {{ col }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, ri) in spec.rows" :key="ri">
            <td v-for="col in spec.columns ?? Object.keys(row)" :key="col">
              {{ row[col] }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.chart-wrap {
  margin: 12px 0;
  border: 1px solid var(--ds-input-border);
  border-radius: 8px;
  padding: 8px;
  background: #fff;
}
.chart {
  width: 100%;
  height: 320px;
}
.table-wrap {
  margin: 12px 0;
}
.table-title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 6px;
  color: var(--ds-text-secondary);
}
.table-scroll {
  overflow: auto;
  max-height: 280px;
  border: 1px solid var(--ds-input-border);
  border-radius: 8px;
}
table {
  border-collapse: collapse;
  width: 100%;
  font-size: 13px;
}
th,
td {
  border-bottom: 1px solid var(--ds-input-border);
  padding: 8px 10px;
  text-align: left;
  white-space: nowrap;
}
th {
  background: #f9fafb;
  position: sticky;
  top: 0;
}
</style>
