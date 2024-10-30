@php
    /** @var \App\Models\Session $session */
    /** @var \App\Models\SessionServiceSuiteTest[] $tests */
@endphp

<div>
<div x-data="{open: @entangle('open')}"
     x-on:run-created="$refs.dialog.close()"
    >
    <button type="button" class="filled" @click="$dispatch('show')">Run Manual Tests</button>
    <dialog x-ref="dialog" class="dialog" x-bind:open="open">
        <button type="submit" class="dialog-close" aria-label="close" @click="$dispatch('close')">X</button>

        <div>Test {{ $index + 1}}/{{$tests->count()}}</div>
        @if ($this->currentTest)
            <livewire:runs.manual-test-form key="{{$this->currentTest->id}}" :test="$this->currentTest" />
        @else
            <p>No test selected</p>
        @endif

        @if ($tests->count() > 1)
            <div>
                @foreach ($tests as $test)
                    <button type="button"  wire:click="goToIndex({{ $loop->index }})">
                        {{ $loop->index + 1 }}
                    </button>
                @endforeach
            </div>
        @endif
    </dialog>
</div>
</div>

