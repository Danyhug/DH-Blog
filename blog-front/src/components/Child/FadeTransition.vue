<!--
  Shared by every swapped view so timing lives in one place; out-in keeps leave short to avoid a blank gap.
  Kept outside <template>: a root-level comment survives in dev and turns the root into a fragment.
-->
<template>
  <Transition mode="out-in"
    enter-active-class="transition-[opacity,translate] duration-300 ease-out"
    enter-from-class="opacity-0 translate-y-3"
    leave-active-class="transition-[opacity,translate] duration-150 ease-in"
    leave-to-class="opacity-0 -translate-y-1">
    <slot />
  </Transition>
</template>
