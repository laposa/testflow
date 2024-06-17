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
                <input type="text"
                       id="name"
                       name="name"
                       required
                       placeholder="Session name"
                       value="{{ old('name') }}"
                >
                <select name="assignees" id="">
                    <option value="" disabled selected>Assign to</option>
                    <option value="1">User 1</option>
                    <option value="2">User 2</option>
                    <option value="3">User 3</option>
                </select>
            </div>
        </x-portal-section>
        <x-portal-section title="Test suites" width="full">
            <x-test-suites.list :select="true" :suites="$suites"/>
        </x-portal-section>

        <button type="submit" class="button" >Create</button>
    </form>
</x-layout>
