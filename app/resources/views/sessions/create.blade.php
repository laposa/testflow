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
                <input style="width:100%" type="text"
                       id="name"
                       name="name"
                       required
                       placeholder="Session name"
                       value="{{ old('name') }}"
                >
            </div>
        </x-portal-section>
        <x-portal-section title="Test suites" width="full">
            <x-test-suites.select :select="true" :suites="$suites"/>
        </x-portal-section>

        <button type="submit" class="button" >Create</button>
    </form>
</x-layout>
