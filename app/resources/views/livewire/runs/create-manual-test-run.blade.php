@php
    /** @var \App\Models\Session $session */
    /** @var \App\Models\SessionServiceSuiteTest[] $tests */
@endphp

<div>
<div x-data="{
    open: @entangle('open'),
    close() {
        if (confirm('Closing will finish the run, would you like to continue?')) {
            $dispatch('end-run');
        }
    }
}">
    <button type="button" class="filled" wire:click="startRun()">Run Manual Tests ({{ $service->name }})</button>
    <dialog x-ref="dialog" class="dialog" x-bind:open="open">
        <button type="submit" class="dialog-close" aria-label="close" @click="close()">X</button>

        <div>Test {{ $index + 1}}/{{$tests->count()}}</div>
        @if ($this->currentTest)
            <livewire:runs.manual-test-form
                key="{{$this->currentTest->id}}"
                :test="$this->currentTest"
                @updated="updateResults($event.detail.id, $event.detail.result, $event.detail.comment)"/>
        @else
            <p>No test selected</p>
        @endif
    </dialog>
</div>
</div>

