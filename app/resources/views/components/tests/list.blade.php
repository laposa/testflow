@props(['select' => false, 'link' => true, 'tests' => []])

<table
    @class([
        'select' => $select
    ])>
    <thead>
    <tr>
       @if ($select)
        <th>
            <label for="selector-all" class="checkbox all">
            <input type="checkbox" id="selector-all">
            <span class="checkmark"></span>
            </label>
        </th>
        @endif
        <th>Test</th>
        <th>Type</th>
        <th>Last result</th>
        <th>Error</th>
        <th>Last run</th>
    </tr>
    </thead>
    <tbody>
    @foreach($tests as $test)
    <tr>
        @if ($select)
        <td>
            <label for="selector-test-{{ $test->id }}" class="checkbox">
            <input type="checkbox" name="manual" id="selector-test-{{ $test->id }}">
            <span class="checkmark"></span>
            </label>
        </td>
        @endif
        <td>
            @if ($link)
                <a href="/tests/{{ $test->id }}">{{ $test->title }}</a>
            @else
                <span>{{$test->title}}</span>
            @endif
        </td>
        <td>
            @if ($test->type === \App\Enums\TestTypeEnum::automated->value)
                <span>Automated</span>
            @else
                <span>Manual</span>
            @endif
        </td>
        <td>
            @switch($test->lastResult)
                @case(\App\Enums\TestResultEnum::pass->value)
                    <span class="pass">Passed</span>
                    @break
                @case(\App\Enums\TestResultEnum::fail->value)
                    <span class="fail">Failed</span>
                    @break
                @case(\App\Enums\TestResultEnum::notRun->value)
                    <span class="fresh">Not Run</span>
                    @break
            @endswitch
        </td>
        <td>{{$test->error}}</td>
        <td>
            @if ($test->timestamp)
                <a href={"sessions/1/test/{{ $test->id }}/run/{{ $test->id }}">
                    {{ now()->format('Y-m-d H:i:s') }}
                </a>
            @else

            @endif
        </td>
    </tr>
  @endforeach
    </tbody>
</table>
