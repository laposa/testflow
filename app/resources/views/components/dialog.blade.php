@props(['name', 'show' => false ])


    <div class="modal"
         x-cloak
         x-data="{
            show: @js($show),
            open(name) {
                this.show = true;
                this.$dispatch('modal-opened', name);
            },
            close(name) {
                this.show = false;
                this.$dispatch('modal-closed', name);
            },
        }"
         x-init="
            $watch('show', (value) => {
                if (value) {
                    document.body.classList.add('overflow-hidden')
                } else {
                    document.body.classList.remove('overflow-hidden')
                }
            })
        "
         x-show="show"
        {{ $attributes->merge([
            "x-on:open-modal.window" => "\$event.detail == '{$name}' ? open('{$name}') : null",
            "x-on:close-modal.window" => "\$event.detail == '{$name}' ? close('{$name}') : null",
            "x-on:keydown.escape.window" => "\$dispatch('close-modal','{$name}')",
        ]) }}>
        <div class="modal-content" @click.outside="$dispatch('close-modal', '{{ $name }}')">
            <button @click.prevent="$dispatch('close-modal','{{ $name }}')">×</button>
            <div class="content">
                {{ $slot }}
            </div>
        </div>
    </div>

