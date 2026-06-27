<template>
  <div class="not-found bg-white dark:bg-[#1e1e1e]">
    <div class="not-found__content">
      <img src="/assets/logo.svg" class="not-found__logo" alt="DeguBase" />
      <h1 class="not-found__title text-text-1">404 — No base of degus here</h1>
      <p class="not-found__message text-text-2">
        This is not the degu you are searching for. The page may have been moved, deleted, or never existed.
      </p>
      <router-link to="/" class="not-found__link">Go back home</router-link>
    </div>
  </div>
</template>

<style scoped>
.not-found {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 2rem;
}

.not-found__content {
  text-align: center;
  max-width: 400px;
}

.not-found__logo {
  display: block;
  width: 96px;
  height: 96px;
  margin: 0 auto 1.5rem;
}

.not-found__title {
  font-size: 1.75rem;
  font-weight: 600;
  margin-bottom: 0.75rem;
}

.not-found__message {
  margin-bottom: 1.5rem;
  line-height: 1.6;
}

.not-found__link {
  display: inline-block;
  padding: 0.5rem 1.25rem;
  background: var(--color-brand-600);
  color: white;
  border-radius: 6px;
  text-decoration: none;
  font-weight: 500;
  transition: opacity 0.15s;
}

.not-found__link:hover {
  opacity: 0.85;
}
</style>
