document.addEventListener('alpine:init', () => {
  Alpine.data('email', () => ({
    advanced: {
      show: false,

      toggle() {
        this.advanced.show = !this.advanced.show;
      },
    },
  }))
})

