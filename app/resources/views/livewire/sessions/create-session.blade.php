@php
    /**
     * @var array $suites
     */
@endphp

<div>
    <form action="/sessions" method="POST">
        @csrf

        <x-portal-section title="Create new session">
            <div class="information">
                <input style="width:60%" type="text" id="name" name="name" required
                    placeholder="Session name" value="{{ old('name') }}">
                <select name="environment">
                    <option value="preprod">PreProd environment</option>
                </select>
            </div>
        </x-portal-section>

        <x-portal-section>
            <x-test-suites.list-interactive :suites="$suites" />
        </x-portal-section>

        <x-portal-section>
            <button type="submit" class="button filled">Create</button>
        </x-portal-section>
    </form>
</div>
