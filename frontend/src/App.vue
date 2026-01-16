<template>
  <div class="container py-5">
    <div class="row justify-content-center">
      <div class="col-md-8 text-center">
        <div class="mb-4">
          <i class="ti ti-database-cog fs-1 text-primary"></i>
        </div>
        <h1 class="display-4 fw-bold mb-3">Celerix Store Cockpit</h1>
        <p class="lead text-muted mb-5">
          Welcome to the management interface for your Liquid Data engine. 
          The Persona UI is currently under construction.
        </p>
        
        <div class="card shadow-sm border-0 bg-light p-4 mb-4">
          <div class="row text-start g-4">
            <div class="col-md-6">
              <h5 class="fw-bold"><i class="ti ti-plug me-2"></i>Engine Port</h5>
              <p class="text-muted small">7001 (TCP/TLS)</p>
            </div>
            <div class="col-md-6">
              <h5 class="fw-bold"><i class="ti ti-world me-2"></i>HTTP API</h5>
              <p class="text-muted small">7002 (JSON)</p>
            </div>
          </div>
        </div>

        <div class="d-flex justify-content-center gap-3">
          <button class="btn btn-primary" @click="fetchStats">
            <i class="ti ti-refresh me-1"></i> Check Engine Status
          </button>
        </div>
        
        <div v-if="stats" class="mt-4 animate-in">
          <span class="badge bg-success-subtle text-success border border-success">
            Engine Active: {{ stats.length }} Personas Loaded
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';

const stats = ref<any[] | null>(null);

const fetchStats = async () => {
  try {
    const response = await fetch('/api/personas');
    if (response.ok) {
      stats.value = await response.json();
    }
  } catch (e) {
    console.error('Failed to fetch stats', e);
  }
};
</script>

<style>
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;600;700&display=swap');

body {
  font-family: 'Inter', sans-serif;
  background-color: #f8f9fa;
}

.animate-in {
  animation: fadeIn 0.5s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
