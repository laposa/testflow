<x-layout>
    <form action="/sessions" method="POST">
        @if ($errors->any())
        <div class="error-list">
            <ul>
                @foreach ($errors->all() as $error)
                <li>{{ $error }}</li>
                @endforeach
            </ul>
        </div>
        @endif
        @csrf
        <x-portal-section title="Create new session" width="full">
            <div class="information">
                <input style="width:60%" type="text" id="name" name="name" required placeholder="Session name" value="{{ old('name') }}">
                <select name="environment">
                    <option value="prod">Prod environment</option>
                    <option value="beta">Beta environment</option>
                    <option value="preprod">PreProd environment</option>
                    <option value="dev">Dev environment</option>
                </select>
            </div>
        </x-portal-section>
        <x-portal-section width="full">
            <x-test-suites.select :select="true" :suites="$suites" />
        </x-portal-section>

        <x-portal-section width="full">
            <button type="submit" class="button">Create</button>
        </x-portal-section>

        <x-portal-section width="full">
            <x-test-suites.select title="Suites Without Workflow" :select="false" :suites="$suitesWithoutWorkflow" />
        </x-portal-section>
    </form>
</x-layout>