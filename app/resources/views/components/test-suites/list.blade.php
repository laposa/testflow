@props(['suites' => []])
<table @class([])>
    <tbody>
        
        @foreach($suites as $suite)
        <tr>
            <td>{{str_replace(['laposa/', 'tests/'], '', $suite['path'])}}</td>
            <td>
                <!-- <span class="expand" data-target="selector-suite-tests-{{ $suite['repository_id'] }}">Expand</span> -->
            </td>
        </tr>
        <tr id="selector-suite-tests-{{ $suite['repository_id'] }}" class="collapsible">
            <td colspan="2">
            </td>
        </tr>
        @endforeach
    </tbody>
</table>